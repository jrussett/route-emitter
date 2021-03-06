package routehandlers

import (
	"errors"

	"code.cloudfoundry.org/bbs/models"
	loggingclient "code.cloudfoundry.org/diego-logging-client"
	"code.cloudfoundry.org/lager"
	"code.cloudfoundry.org/route-emitter/emitter"
	"code.cloudfoundry.org/route-emitter/routingtable"
	"code.cloudfoundry.org/route-emitter/watcher"
)

const (
	routesTotalMetric         = "RoutesTotal"
	routesSyncedCounter       = "RoutesSynced"
	routesRegisteredCounter   = "RoutesRegistered"
	routesUnregisteredCounter = "RoutesUnregistered"
	httpRouteCount            = "HTTPRouteCount"
	tcpRouteCount             = "TCPRouteCount"
)

type Handler struct {
	routingTable      routingtable.RoutingTable
	natsEmitter       emitter.NATSEmitter
	routingAPIEmitter emitter.RoutingAPIEmitter
	localMode         bool
	metronClient      loggingclient.IngressClient
}

var _ watcher.RouteHandler = new(Handler)

func NewHandler(routingTable routingtable.RoutingTable, natsEmitter emitter.NATSEmitter, routingAPIEmitter emitter.RoutingAPIEmitter, localMode bool, metronClient loggingclient.IngressClient) *Handler {
	return &Handler{
		routingTable:      routingTable,
		natsEmitter:       natsEmitter,
		routingAPIEmitter: routingAPIEmitter,
		localMode:         localMode,
		metronClient:      metronClient,
	}
}

func (handler *Handler) HandleEvent(logger lager.Logger, event models.Event) {

	switch event := event.(type) {
	case *models.DesiredLRPCreatedEvent:
		desiredInfo := event.DesiredLrp.DesiredLRPSchedulingInfo()
		handler.handleDesiredCreate(logger, &desiredInfo)
	case *models.DesiredLRPChangedEvent:
		before := event.Before.DesiredLRPSchedulingInfo()
		after := event.After.DesiredLRPSchedulingInfo()
		handler.handleDesiredUpdate(logger, &before, &after)
	case *models.DesiredLRPRemovedEvent:
		desiredInfo := event.DesiredLrp.DesiredLRPSchedulingInfo()
		handler.handleDesiredDelete(logger, &desiredInfo)
	case *models.ActualLRPInstanceCreatedEvent:
		if event.ActualLrp == nil {
			logger.Error("nil-actual-lrp", nil, lager.Data{"event-type": event.EventType()})
			return
		}
		handler.handleActualCreate(logger, event.ActualLrp)
	case *models.ActualLRPInstanceChangedEvent:
		before := event.Before.ToActualLRP(event.ActualLRPKey, event.ActualLRPInstanceKey)
		after := event.After.ToActualLRP(event.ActualLRPKey, event.ActualLRPInstanceKey)
		logger.Debug("received-actual-lrp-changed-event", lager.Data{"before": before, "after": after})
		if before == nil || after == nil {
			logger.Error("nil-actual-lrp", nil, lager.Data{"event-type": event.EventType()})
			return
		}
		handler.handleActualUpdate(logger, before, after)
	case *models.ActualLRPInstanceRemovedEvent:
		logger.Debug("received-actual-lrp-instance-removed-event", lager.Data{"lrp": event.ActualLrp})
		if event.ActualLrp == nil {
			logger.Error("nil-actual-lrp", nil, lager.Data{"event-type": event.EventType()})
			return
		}
		handler.handleActualDelete(logger, event.ActualLrp)
	default:
		logger.Error("did-not-handle-unrecognizable-event", errors.New("unrecognizable-event"), lager.Data{"event-type": event.EventType()})
	}
}

func (handler *Handler) EmitExternal(logger lager.Logger) {
	routingEvents, messagesToEmit := handler.routingTable.GetExternalRoutingEvents()

	logger.Debug("emitting-nats-messages", lager.Data{"messages": messagesToEmit})
	if handler.natsEmitter != nil {
		err := handler.natsEmitter.Emit(messagesToEmit)
		if err != nil {
			logger.Error("failed-to-emit-nats-routes", err)
		}
	}

	logger.Debug("emitting-routing-api-messages", lager.Data{"messages": routingEvents})
	if handler.routingAPIEmitter != nil {
		err := handler.routingAPIEmitter.Emit(routingEvents)
		if err != nil {
			logger.Error("failed-to-emit-tcp-routes", err)
		}
	}

	err := handler.metronClient.IncrementCounterWithDelta(routesSyncedCounter, messagesToEmit.RouteRegistrationCount())
	if err != nil {
		logger.Error("failed-send-routes-synced-count-metric", err)
	}
	err = handler.metronClient.SendMetric(routesTotalMetric, handler.routingTable.HTTPAssociationsCount())
	if err != nil {
		logger.Error("failed-to-send-total-route-count-metric", err)
	}
}

func (handler *Handler) EmitInternal(logger lager.Logger) {
	_, messagesToEmit := handler.routingTable.GetInternalRoutingEvents()

	logger.Debug("emitting-nats-messages", lager.Data{"messages": messagesToEmit})
	if handler.natsEmitter != nil {
		err := handler.natsEmitter.Emit(messagesToEmit)
		if err != nil {
			logger.Error("failed-to-emit-nats-routes", err)
		}
	}
}

func (handler *Handler) Sync(
	logger lager.Logger,
	desired []*models.DesiredLRPSchedulingInfo,
	actuals []*models.ActualLRP,
	domains models.DomainSet,
	cachedEvents map[string]models.Event,
) {
	logger = logger.Session("sync")
	logger.Debug("starting")
	defer logger.Debug("completed")

	nullLogger := lager.NewLogger("") // ignore log messsages from the routing table
	newTable := routingtable.NewRoutingTable(false, handler.metronClient)

	for _, lrp := range desired {
		newTable.SetRoutes(nullLogger, nil, lrp)
	}

	for _, lrp := range actuals {
		newTable.AddEndpoint(nullLogger, lrp)
	}

	natsEmitter := handler.natsEmitter
	routingAPIEmitter := handler.routingAPIEmitter
	table := handler.routingTable

	handler.natsEmitter = nil
	handler.routingAPIEmitter = nil
	handler.routingTable = newTable

	for _, event := range cachedEvents {
		handler.HandleEvent(logger, event)
	}

	handler.routingTable = table
	handler.natsEmitter = natsEmitter
	handler.routingAPIEmitter = routingAPIEmitter

	routeMappings, messages := handler.routingTable.Swap(nullLogger, newTable, domains)
	logger.Debug("start-emitting-messages", lager.Data{
		"num-registration-messages":            len(messages.RegistrationMessages),
		"num-unregistration-messages":          len(messages.UnregistrationMessages),
		"num-internal-registration-messages":   len(messages.InternalRegistrationMessages),
		"num-internal-unregistration-messages": len(messages.InternalUnregistrationMessages),
	})
	handler.emitMessages(logger, messages, routeMappings)
	logger.Debug("done-emitting-messages", lager.Data{
		"num-registration-messages":            len(messages.RegistrationMessages),
		"num-unregistration-messages":          len(messages.UnregistrationMessages),
		"num-internal-registration-messages":   len(messages.InternalRegistrationMessages),
		"num-internal-unregistration-messages": len(messages.InternalUnregistrationMessages),
	})

	if handler.localMode {
		err := handler.metronClient.SendMetric(httpRouteCount, handler.routingTable.HTTPAssociationsCount())
		if err != nil {
			logger.Error("failed-to-send-http-routes-count-metric", err)
		}
		err = handler.metronClient.SendMetric(tcpRouteCount, handler.routingTable.TCPAssociationsCount())
		if err != nil {
			logger.Error("failed-to-send-tcp-route-count-metric", err)
		}
	}
}

func (handler *Handler) RefreshDesired(logger lager.Logger, desiredInfo []*models.DesiredLRPSchedulingInfo) {
	for _, desiredLRP := range desiredInfo {
		routeMappings, messagesToEmit := handler.routingTable.SetRoutes(logger, nil, desiredLRP)
		handler.emitMessages(logger, messagesToEmit, routeMappings)
	}
}

func (handler *Handler) ShouldRefreshDesired(actualLRP *models.ActualLRP) bool {
	return !handler.routingTable.HasExternalRoutes(actualLRP)
}

func (handler *Handler) handleDesiredCreate(logger lager.Logger, desiredLRP *models.DesiredLRPSchedulingInfo) {
	routeMappings, messagesToEmit := handler.routingTable.SetRoutes(logger, nil, desiredLRP)
	handler.emitMessages(logger, messagesToEmit, routeMappings)
}

func (handler *Handler) handleDesiredUpdate(logger lager.Logger, before, after *models.DesiredLRPSchedulingInfo) {
	routeMappings, messagesToEmit := handler.routingTable.SetRoutes(logger, before, after)
	handler.emitMessages(logger, messagesToEmit, routeMappings)
}

func (handler *Handler) handleDesiredDelete(logger lager.Logger, schedulingInfo *models.DesiredLRPSchedulingInfo) {
	routeMappings, messagesToEmit := handler.routingTable.RemoveRoutes(logger, schedulingInfo)
	handler.emitMessages(logger, messagesToEmit, routeMappings)
}

func (handler *Handler) handleActualCreate(logger lager.Logger, actualLRP *models.ActualLRP) {
	if actualLRP.State != models.ActualLRPStateRunning {
		return
	}
	routeMappings, messagesToEmit := handler.routingTable.AddEndpoint(logger, actualLRP)
	handler.emitMessages(logger, messagesToEmit, routeMappings)
}

func (handler *Handler) handleActualUpdate(logger lager.Logger, before, after *models.ActualLRP) {
	var (
		messagesToEmit routingtable.MessagesToEmit
		routeMappings  routingtable.TCPRouteMappings
	)
	switch {
	case after.State == models.ActualLRPStateRunning:
		routeMappings, messagesToEmit = handler.routingTable.AddEndpoint(logger, after)
		if before.State == models.ActualLRPStateRunning &&
			before.Presence == models.ActualLRP_Ordinary && after.Presence == models.ActualLRP_Evacuating {
			removeRouteMappings, removeMessagesToEmit := handler.routingTable.RemoveEndpoint(logger, before)
			routeMappings = routeMappings.Merge(removeRouteMappings)
			messagesToEmit = messagesToEmit.Merge(removeMessagesToEmit)
		}
	case before.State == models.ActualLRPStateRunning && after.State != models.ActualLRPStateRunning:
		routeMappings, messagesToEmit = handler.routingTable.RemoveEndpoint(logger, before)
	}
	handler.emitMessages(logger, messagesToEmit, routeMappings)
}

func (handler *Handler) handleActualDelete(logger lager.Logger, actualLRP *models.ActualLRP) {
	if actualLRP == nil || actualLRP.State != models.ActualLRPStateRunning {
		return
	}
	routeMappings, messagesToEmit := handler.routingTable.RemoveEndpoint(logger, actualLRP)
	handler.emitMessages(logger, messagesToEmit, routeMappings)
}

func (handler *Handler) emitMessages(logger lager.Logger, messagesToEmit routingtable.MessagesToEmit, routeMappings routingtable.TCPRouteMappings) {
	if handler.natsEmitter != nil {
		logger.Debug("emit-messages", lager.Data{"messages": messagesToEmit})
		err := handler.natsEmitter.Emit(messagesToEmit)
		if err != nil {
			logger.Error("failed-to-emit-http-routes", err)
		}
		err = handler.metronClient.IncrementCounterWithDelta(routesRegisteredCounter, messagesToEmit.RouteRegistrationCount())
		if err != nil {
			logger.Error("failed-to-emit-registration-message-count", err)
		}
		err = handler.metronClient.IncrementCounterWithDelta(routesUnregisteredCounter, messagesToEmit.RouteUnregistrationCount())
		if err != nil {
			logger.Error("failed-to-emit-unregistration-message-count", err)
		}
	} else {
		logger.Info("no-emitter-configured-skipping-emit-messages", lager.Data{"messages": messagesToEmit})
	}

	if handler.routingAPIEmitter != nil {
		err := handler.routingAPIEmitter.Emit(routeMappings)
		if err != nil {
			logger.Error("failed-to-emit-http-routes", err)
		}
	}
}
