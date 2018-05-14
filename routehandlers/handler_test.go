package routehandlers_test

import (
	"encoding/json"
	"fmt"

	mfakes "code.cloudfoundry.org/diego-logging-client/testhelpers"
	loggregator "code.cloudfoundry.org/go-loggregator"

	"code.cloudfoundry.org/bbs/models"
	"code.cloudfoundry.org/lager/lagertest"
	"code.cloudfoundry.org/route-emitter/emitter/fakes"
	"code.cloudfoundry.org/route-emitter/routehandlers"
	"code.cloudfoundry.org/route-emitter/routingtable"
	"code.cloudfoundry.org/route-emitter/routingtable/fakeroutingtable"
	"code.cloudfoundry.org/routing-info/cfroutes"
	"github.com/gogo/protobuf/proto"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"
)

const logGuid = "some-log-guid"

type randomEvent struct {
	proto.Message
}

func (e randomEvent) EventType() string {
	return "random"
}
func (e randomEvent) Key() string {
	return "random"
}

var _ = Describe("Handler", func() {
	type counter struct {
		name  string
		delta uint64
	}
	type metric struct {
		name  string
		value int
	}

	const (
		expectedDomain                  = "domain"
		expectedProcessGuid             = "process-guid"
		expectedInstanceGUID            = "instance-guid"
		expectedIndex                   = 0
		expectedHost                    = "1.1.1.1"
		expectedInstanceAddress         = "2.2.2.2"
		expectedExternalPort            = 11000
		expectedAdditionalExternalPort  = 22000
		expectedContainerPort           = 11
		expectedAdditionalContainerPort = 22
		expectedRouteServiceUrl         = "https://so.good.com"
	)

	var (
		fakeTable   *fakeroutingtable.FakeRoutingTable
		natsEmitter *fakes.FakeNATSEmitter

		expectedRoutes  []string
		expectedCFRoute cfroutes.CFRoute

		dummyMessagesToEmit routingtable.MessagesToEmit
		fakeMetronClient    *mfakes.FakeIngressClient

		logger *lagertest.TestLogger

		routeHandler *routehandlers.Handler

		emptyTCPRouteMappings routingtable.TCPRouteMappings

		counterChan chan counter
		metricChan  chan metric
	)

	BeforeEach(func() {
		fakeTable = &fakeroutingtable.FakeRoutingTable{}
		natsEmitter = &fakes.FakeNATSEmitter{}
		logger = lagertest.NewTestLogger("test")

		dummyEndpoint := routingtable.Endpoint{
			InstanceGUID: expectedInstanceGUID,
			Index:        expectedIndex,
			Host:         expectedHost,
			Port:         expectedContainerPort,
		}
		dummyMessageFoo := routingtable.RegistryMessageFor(dummyEndpoint, routingtable.Route{Hostname: "foo.com", LogGUID: logGuid}, true)
		dummyMessageBar := routingtable.RegistryMessageFor(dummyEndpoint, routingtable.Route{Hostname: "bar.com", LogGUID: logGuid}, true)
		dummyMessagesToEmit = routingtable.MessagesToEmit{
			RegistrationMessages: []routingtable.RegistryMessage{dummyMessageFoo, dummyMessageBar},
		}

		expectedRoutes = []string{"route-1", "route-2"}
		expectedCFRoute = cfroutes.CFRoute{Hostnames: expectedRoutes, Port: expectedContainerPort, RouteServiceUrl: expectedRouteServiceUrl}

		fakeMetronClient = &mfakes.FakeIngressClient{}
		counterChan = make(chan counter, 10)
		fakeMetronClient.IncrementCounterWithDeltaStub = func(name string, delta uint64) error {
			counterChan <- counter{name: name, delta: delta}
			return nil
		}
		metricChan = make(chan metric, 10)
		fakeMetronClient.SendMetricStub = func(name string, value int, opts ...loggregator.EmitGaugeOption) error {
			metricChan <- metric{name: name, value: value}
			return nil
		}

		routeHandler = routehandlers.NewHandler(fakeTable, natsEmitter, nil, false, fakeMetronClient)
	})

	Context("when an unrecognized event is received", func() {
		It("logs an error", func() {
			routeHandler.HandleEvent(logger, randomEvent{})
			Expect(logger).To(gbytes.Say("did-not-handle-unrecognizable-event"))
		})
	})

	Describe("DesiredLRP Event", func() {
		Context("DesiredLRPCreated Event", func() {
			var (
				desiredLRP *models.DesiredLRP
			)

			BeforeEach(func() {
				routes := cfroutes.CFRoutes{expectedCFRoute}.RoutingInfo()
				desiredLRP = &models.DesiredLRP{
					Action: models.WrapAction(&models.RunAction{
						User: "me",
						Path: "ls",
					}),
					Domain:      "tests",
					ProcessGuid: expectedProcessGuid,
					Ports:       []uint32{expectedContainerPort},
					Routes:      &routes,
					LogGuid:     logGuid,
				}
				fakeTable.SetRoutesReturns(emptyTCPRouteMappings, dummyMessagesToEmit)
			})

			JustBeforeEach(func() {
				routeHandler.HandleEvent(logger, models.NewDesiredLRPCreatedEvent(desiredLRP))
			})

			It("should set the routes on the table", func() {
				Expect(fakeTable.SetRoutesCallCount()).To(Equal(1))
				before, after := fakeTable.SetRoutesArgsForCall(0)
				Expect(before).To(BeNil())
				Expect(*after).To(Equal(desiredLRP.DesiredLRPSchedulingInfo()))
			})

			It("sends a 'routes registered' metric", func() {
				Eventually(counterChan).Should(Receive(Equal(counter{
					name:  "RoutesRegistered",
					delta: 2,
				})))
			})

			It("sends a 'routes unregistered' metric", func() {
				Eventually(counterChan).Should(Receive(Equal(counter{
					name:  "RoutesUnregistered",
					delta: 0,
				})))
			})

			It("should emit whatever the table tells it to emit", func() {
				Expect(natsEmitter.EmitCallCount()).To(Equal(1))
				messagesToEmit := natsEmitter.EmitArgsForCall(0)
				Expect(messagesToEmit).To(Equal(dummyMessagesToEmit))
			})

			Context("when there are diego ssh-keys on the route", func() {
				BeforeEach(func() {
					diegoSSHInfo := json.RawMessage([]byte(`{"ssh-key": "ssh-value"}`))

					routes := cfroutes.CFRoutes{expectedCFRoute}.RoutingInfo()
					routes["diego-ssh"] = &diegoSSHInfo

					desiredLRP.Routes = &routes
				})

				It("does not log anything", func() {
					Expect(fakeTable.SetRoutesCallCount()).To(Equal(1))
					Expect(logger.Buffer()).NotTo(gbytes.Say("diego-ssh"))
				})
			})
		})

		Context("DesiredLRPChanged Event", func() {
			type metric struct {
				name  string
				delta uint64
			}

			var (
				originalDesiredLRP, changedDesiredLRP *models.DesiredLRP
				metricChan                            chan metric
			)

			BeforeEach(func() {
				fakeTable.SetRoutesReturns(emptyTCPRouteMappings, dummyMessagesToEmit)
				routes := cfroutes.CFRoutes{{Hostnames: expectedRoutes, Port: expectedContainerPort}}.RoutingInfo()

				originalDesiredLRP = &models.DesiredLRP{
					Action: models.WrapAction(&models.RunAction{
						User: "me",
						Path: "ls",
					}),
					Domain:      "tests",
					ProcessGuid: expectedProcessGuid,
					LogGuid:     logGuid,
					Routes:      &routes,
					Instances:   3,
				}
				changedDesiredLRP = &models.DesiredLRP{
					Action: models.WrapAction(&models.RunAction{
						User: "me",
						Path: "ls",
					}),
					Domain:          "tests",
					ProcessGuid:     expectedProcessGuid,
					LogGuid:         logGuid,
					Routes:          &routes,
					ModificationTag: &models.ModificationTag{Epoch: "abcd", Index: 1},
					Instances:       3,
				}
				metricChan = make(chan metric, 10)
				fakeMetronClient.IncrementCounterWithDeltaStub = func(name string, delta uint64) error {
					metricChan <- metric{name: name, delta: delta}
					return nil
				}
			})

			JustBeforeEach(func() {
				routeHandler.HandleEvent(logger, models.NewDesiredLRPChangedEvent(originalDesiredLRP, changedDesiredLRP))
			})

			It("should set the routes on the table", func() {
				Expect(fakeTable.SetRoutesCallCount()).To(Equal(1))
				before, after := fakeTable.SetRoutesArgsForCall(0)
				Expect(*before).To(Equal(originalDesiredLRP.DesiredLRPSchedulingInfo()))
				Expect(*after).To(Equal(changedDesiredLRP.DesiredLRPSchedulingInfo()))
			})

			It("sends a 'routes registered' metric", func() {
				Eventually(metricChan).Should(Receive(Equal(metric{
					name:  "RoutesRegistered",
					delta: 2,
				})))
			})

			It("sends a 'routes unregistered' metric", func() {
				Eventually(metricChan).Should(Receive(Equal(metric{
					name:  "RoutesUnregistered",
					delta: 0,
				})))
			})

			It("should emit whatever the table tells it to emit", func() {
				Expect(natsEmitter.EmitCallCount()).To(Equal(1))
				messagesToEmit := natsEmitter.EmitArgsForCall(0)
				Expect(messagesToEmit).To(Equal(dummyMessagesToEmit))
			})

			Context("when there are diego ssh-keys on the route", func() {
				BeforeEach(func() {
					diegoSSHInfo := json.RawMessage([]byte(`{"ssh-key": "ssh-value"}`))

					routes := cfroutes.CFRoutes{expectedCFRoute}.RoutingInfo()
					routes["diego-ssh"] = &diegoSSHInfo

					changedDesiredLRP.Routes = &routes
				})

				It("does not log them", func() {
					Expect(fakeTable.SetRoutesCallCount()).To(Equal(1))
					Expect(logger.Buffer()).NotTo(gbytes.Say("diego-ssh"))
				})
			})
		})

		Context("when a delete event occurs", func() {
			var desiredLRP *models.DesiredLRP

			BeforeEach(func() {
				fakeTable.RemoveRoutesReturns(emptyTCPRouteMappings, dummyMessagesToEmit)
				routes := cfroutes.CFRoutes{expectedCFRoute}.RoutingInfo()
				desiredLRP = &models.DesiredLRP{
					Action: models.WrapAction(&models.RunAction{
						User: "me",
						Path: "ls",
					}),
					Domain:          "tests",
					ProcessGuid:     expectedProcessGuid,
					Ports:           []uint32{expectedContainerPort},
					Routes:          &routes,
					LogGuid:         logGuid,
					ModificationTag: &models.ModificationTag{Epoch: "defg", Index: 2},
				}
			})

			JustBeforeEach(func() {
				routeHandler.HandleEvent(logger, models.NewDesiredLRPRemovedEvent(desiredLRP))
			})

			It("should remove the routes from the table", func() {
				Expect(fakeTable.RemoveRoutesCallCount()).To(Equal(1))
				lrp := fakeTable.RemoveRoutesArgsForCall(0)
				Expect(*lrp).To(Equal(desiredLRP.DesiredLRPSchedulingInfo()))
			})

			It("should emit whatever the table tells it to emit", func() {
				Expect(natsEmitter.EmitCallCount()).To(Equal(1))

				messagesToEmit := natsEmitter.EmitArgsForCall(0)
				Expect(messagesToEmit).To(Equal(dummyMessagesToEmit))
			})

			Context("when there are diego ssh-keys on the route", func() {
				BeforeEach(func() {
					diegoSSHInfo := json.RawMessage([]byte(`{"ssh-key": "ssh-value"}`))

					routes := cfroutes.CFRoutes{expectedCFRoute}.RoutingInfo()
					routes["diego-ssh"] = &diegoSSHInfo

					desiredLRP.Routes = &routes
				})

				It("does not log them", func() {
					Expect(fakeTable.RemoveRoutesCallCount()).To(Equal(1))
					Expect(logger.Buffer()).NotTo(gbytes.Say("diego-ssh"))
				})
			})
		})
	})

	Describe("Actual LRP changes", func() {
		Context("when a create event occurs", func() {
			var (
				actualLRP *models.FlattenedActualLRP
			)

			Context("when the resulting LRP is in the RUNNING state", func() {
				BeforeEach(func() {
					actualLRP = &models.FlattenedActualLRP{
						ActualLRPKey:         models.NewActualLRPKey(expectedProcessGuid, expectedIndex, "domain"),
						ActualLRPInstanceKey: models.NewActualLRPInstanceKey(expectedInstanceGUID, "cell-id"),
						ActualLRPInfo: models.ActualLRPInfo{
							ActualLRPNetInfo: models.NewActualLRPNetInfo(
								expectedHost,
								expectedInstanceAddress,
								models.NewPortMapping(expectedExternalPort, expectedContainerPort),
								models.NewPortMapping(expectedExternalPort, expectedAdditionalContainerPort),
							),
							State:          models.ActualLRPStateRunning,
							PlacementState: models.PlacementStateType_Normal,
						},
					}
					fakeTable.AddEndpointReturns(emptyTCPRouteMappings, dummyMessagesToEmit)
				})

				JustBeforeEach(func() {
					routeHandler.HandleEvent(logger, models.NewFlattenedActualLRPCreatedEvent(actualLRP))
				})

				It("should add/update the endpoints on the table", func() {
					Expect(fakeTable.AddEndpointCallCount()).To(Equal(1))
					lrpInfo := fakeTable.AddEndpointArgsForCall(0)
					Expect(lrpInfo).To(Equal(actualLRP))
				})

				It("should emit whatever the table tells it to emit", func() {
					Expect(natsEmitter.EmitCallCount()).To(Equal(1))

					messagesToEmit := natsEmitter.EmitArgsForCall(0)
					Expect(messagesToEmit).To(Equal(dummyMessagesToEmit))
				})

				It("sends a 'routes registered' metric", func() {
					Eventually(counterChan).Should(Receive(Equal(counter{
						name:  "RoutesRegistered",
						delta: 2,
					})))
				})

				It("sends a 'routes unregistered' metric", func() {
					Eventually(counterChan).Should(Receive(Equal(counter{
						name:  "RoutesUnregistered",
						delta: 0,
					})))
				})
			})

			Context("when the resulting LRP is not in the RUNNING state", func() {
				JustBeforeEach(func() {
					actualLRP = &models.FlattenedActualLRP{
						ActualLRPKey:         models.NewActualLRPKey(expectedProcessGuid, expectedIndex, "domain"),
						ActualLRPInstanceKey: models.NewActualLRPInstanceKey(expectedInstanceGUID, "cell-id"),
						ActualLRPInfo: models.ActualLRPInfo{
							ActualLRPNetInfo: models.NewActualLRPNetInfo(
								expectedHost,
								expectedInstanceAddress,
								models.NewPortMapping(expectedExternalPort, expectedContainerPort),
								models.NewPortMapping(expectedExternalPort, expectedAdditionalContainerPort),
							),
							State:          models.ActualLRPStateUnclaimed,
							PlacementState: models.PlacementStateType_Normal,
						},
					}
				})

				It("should NOT log the net info", func() {
					Expect(logger).ToNot(gbytes.Say(
						fmt.Sprintf(
							`"net_info":\{"address":"%s","ports":\[\{"container_port":%d,"host_port":%d\},\{"container_port":%d,"host_port":%d\}\]\}`,
							expectedHost,
							expectedContainerPort,
							expectedExternalPort,
							expectedAdditionalContainerPort,
							expectedExternalPort,
						),
					))
				})

				It("doesn't add/update the endpoint on the table", func() {
					Expect(fakeTable.AddEndpointCallCount()).Should(Equal(0))
				})

				It("doesn't emit", func() {
					Expect(natsEmitter.EmitCallCount()).To(Equal(0))
				})
			})
		})

		Context("when a change event occurs", func() {
			Context("when the resulting LRP is in the RUNNING state", func() {
				var (
					afterActualLRP, beforeActualLRP *models.FlattenedActualLRP
				)

				BeforeEach(func() {
					fakeTable.AddEndpointReturns(emptyTCPRouteMappings, dummyMessagesToEmit)

					beforeActualLRP = &models.FlattenedActualLRP{
						ActualLRPKey:         models.NewActualLRPKey(expectedProcessGuid, expectedIndex, "domain"),
						ActualLRPInstanceKey: models.NewActualLRPInstanceKey(expectedInstanceGUID, "cell-id"),
						ActualLRPInfo: models.ActualLRPInfo{
							State:          models.ActualLRPStateClaimed,
							PlacementState: models.PlacementStateType_Normal,
						},
					}
					afterActualLRP = &models.FlattenedActualLRP{
						ActualLRPKey:         models.NewActualLRPKey(expectedProcessGuid, expectedIndex, "domain"),
						ActualLRPInstanceKey: models.NewActualLRPInstanceKey(expectedInstanceGUID, "cell-id"),
						ActualLRPInfo: models.ActualLRPInfo{
							ActualLRPNetInfo: models.NewActualLRPNetInfo(
								expectedHost,
								expectedInstanceAddress,
								models.NewPortMapping(expectedExternalPort, expectedContainerPort),
								models.NewPortMapping(expectedAdditionalExternalPort, expectedAdditionalContainerPort),
							),
							State:          models.ActualLRPStateRunning,
							PlacementState: models.PlacementStateType_Normal,
						},
					}
				})

				JustBeforeEach(func() {
					routeHandler.HandleEvent(logger, models.NewFlattenedActualLRPChangedEvent(beforeActualLRP, afterActualLRP))
				})

				It("should add/update the endpoint on the table", func() {
					Expect(fakeTable.AddEndpointCallCount()).To(Equal(1))

					// Verify the arguments that were passed to AddEndpoint independent of which call was made first.
					type endpointArgs struct {
						key      routingtable.RoutingKey
						endpoint routingtable.Endpoint
					}

					actualLRP := fakeTable.AddEndpointArgsForCall(0)
					Expect(actualLRP).To(Equal(afterActualLRP))
				})

				It("should emit whatever the table tells it to emit", func() {
					Expect(natsEmitter.EmitCallCount()).Should(Equal(1))

					messagesToEmit := natsEmitter.EmitArgsForCall(0)
					Expect(messagesToEmit).To(Equal(dummyMessagesToEmit))
				})

				It("sends a 'routes registered' metric", func() {
					Eventually(counterChan).Should(Receive(Equal(counter{
						name:  "RoutesRegistered",
						delta: 2,
					})))
				})

				It("sends a 'routes unregistered' metric", func() {
					Eventually(counterChan).Should(Receive(Equal(counter{
						name:  "RoutesUnregistered",
						delta: 0,
					})))
				})
			})

			Context("when the resulting LRP transitions away from the RUNNING state", func() {
				var (
					beforeActualLRP, afterActualLRP *models.FlattenedActualLRP
				)

				BeforeEach(func() {
					beforeActualLRP = &models.FlattenedActualLRP{
						ActualLRPKey:         models.NewActualLRPKey(expectedProcessGuid, expectedIndex, "domain"),
						ActualLRPInstanceKey: models.NewActualLRPInstanceKey(expectedInstanceGUID, "cell-id"),
						ActualLRPInfo: models.ActualLRPInfo{
							ActualLRPNetInfo: models.NewActualLRPNetInfo(
								expectedHost,
								expectedInstanceAddress,
								models.NewPortMapping(expectedExternalPort, expectedContainerPort),
								models.NewPortMapping(expectedAdditionalExternalPort, expectedAdditionalContainerPort),
							),
							State:          models.ActualLRPStateRunning,
							PlacementState: models.PlacementStateType_Normal,
						},
					}
					// NOTE: The ActualLRPInstanceKey was be empty in this case, so the BBS would normally just emit
					// a Removed event or something
					afterActualLRP = &models.FlattenedActualLRP{
						ActualLRPKey:         models.NewActualLRPKey(expectedProcessGuid, expectedIndex, "domain"),
						ActualLRPInstanceKey: models.NewActualLRPInstanceKey(expectedInstanceGUID, "cell-id"),
						ActualLRPInfo: models.ActualLRPInfo{
							State:          models.ActualLRPStateUnclaimed,
							PlacementState: models.PlacementStateType_Normal,
						},
					}
					fakeTable.RemoveEndpointReturns(emptyTCPRouteMappings, dummyMessagesToEmit)
				})

				JustBeforeEach(func() {
					routeHandler.HandleEvent(logger, models.NewFlattenedActualLRPChangedEvent(beforeActualLRP, afterActualLRP))
				})

				It("should remove the endpoint from the table", func() {
					Expect(fakeTable.RemoveEndpointCallCount()).To(Equal(1))

					actualLRP := fakeTable.RemoveEndpointArgsForCall(0)
					Expect(actualLRP).To(Equal(beforeActualLRP))
				})

				It("should emit whatever the table tells it to emit", func() {
					Expect(natsEmitter.EmitCallCount()).To(Equal(1))

					messagesToEmit := natsEmitter.EmitArgsForCall(0)
					Expect(messagesToEmit).To(Equal(dummyMessagesToEmit))
				})
			})

			Context("when the endpoint neither starts nor ends in the RUNNING state", func() {
				JustBeforeEach(func() {
					// NOTE: The ActualLRPInstanceKey was be empty in this case, so the BBS would normally just emit
					// an ActualLRPCreated event
					beforeActualLRP := &models.FlattenedActualLRP{
						ActualLRPKey:         models.NewActualLRPKey(expectedProcessGuid, expectedIndex, "domain"),
						ActualLRPInstanceKey: models.NewActualLRPInstanceKey(expectedInstanceGUID, "cell-id"),
						ActualLRPInfo: models.ActualLRPInfo{
							State:          models.ActualLRPStateUnclaimed,
							PlacementState: models.PlacementStateType_Normal,
						},
					}
					afterActualLRP := &models.FlattenedActualLRP{
						ActualLRPKey:         models.NewActualLRPKey(expectedProcessGuid, expectedIndex, "domain"),
						ActualLRPInstanceKey: models.NewActualLRPInstanceKey(expectedInstanceGUID, "cell-id"),
						ActualLRPInfo: models.ActualLRPInfo{
							ActualLRPNetInfo: models.NewActualLRPNetInfo(
								expectedHost,
								expectedInstanceAddress,
								models.NewPortMapping(expectedExternalPort, expectedContainerPort),
								models.NewPortMapping(expectedAdditionalExternalPort, expectedAdditionalContainerPort),
							),
							State:          models.ActualLRPStateClaimed,
							PlacementState: models.PlacementStateType_Normal,
						},
					}
					routeHandler.HandleEvent(logger, models.NewFlattenedActualLRPChangedEvent(beforeActualLRP, afterActualLRP))
				})

				It("should NOT log the net info", func() {
					Expect(logger).ToNot(gbytes.Say(
						fmt.Sprintf(
							`"net_info":\{"address":"%s","ports":\[\{"container_port":%d,"host_port":%d\},\{"container_port":%d,"host_port":%d\}\],"instance_address":"%s"\}`,
							expectedHost,
							expectedContainerPort,
							expectedExternalPort,
							expectedAdditionalContainerPort,
							expectedExternalPort,
							expectedInstanceAddress,
						),
					))
				})

				It("should not remove the endpoint", func() {
					Expect(fakeTable.RemoveEndpointCallCount()).To(BeZero())
				})

				It("should not add or update the endpoint", func() {
					Expect(fakeTable.AddEndpointCallCount()).To(BeZero())
				})
			})

		})

		Context("when a delete event occurs", func() {
			Context("when the actual is in the RUNNING state", func() {
				var (
					actualLRP *models.FlattenedActualLRP
				)

				BeforeEach(func() {
					fakeTable.RemoveEndpointReturns(emptyTCPRouteMappings, dummyMessagesToEmit)

					actualLRP = &models.FlattenedActualLRP{
						ActualLRPKey:         models.NewActualLRPKey(expectedProcessGuid, expectedIndex, "domain"),
						ActualLRPInstanceKey: models.NewActualLRPInstanceKey(expectedInstanceGUID, "cell-id"),
						ActualLRPInfo: models.ActualLRPInfo{
							ActualLRPNetInfo: models.NewActualLRPNetInfo(
								expectedHost,
								expectedInstanceAddress,
								models.NewPortMapping(expectedExternalPort, expectedContainerPort),
								models.NewPortMapping(expectedAdditionalExternalPort, expectedAdditionalContainerPort),
							),
							State:          models.ActualLRPStateRunning,
							PlacementState: models.PlacementStateType_Normal,
						},
					}
				})

				JustBeforeEach(func() {
					routeHandler.HandleEvent(logger, models.NewFlattenedActualLRPRemovedEvent(actualLRP))
				})

				It("should remove the endpoint from the table", func() {
					Expect(fakeTable.RemoveEndpointCallCount()).To(Equal(1))

					lrp := fakeTable.RemoveEndpointArgsForCall(0)
					Expect(lrp).To(Equal(actualLRP))
				})

				It("should emit whatever the table tells it to emit", func() {
					Expect(natsEmitter.EmitCallCount()).To(Equal(1))

					messagesToEmit := natsEmitter.EmitArgsForCall(0)
					Expect(messagesToEmit).To(Equal(dummyMessagesToEmit))
				})
			})

			Context("when the actual is not in the RUNNING state", func() {
				JustBeforeEach(func() {
					actualLRP := &models.FlattenedActualLRP{
						ActualLRPKey: models.NewActualLRPKey(expectedProcessGuid, expectedIndex, "domain"),
						ActualLRPInfo: models.ActualLRPInfo{
							ActualLRPNetInfo: models.NewActualLRPNetInfo(
								expectedHost,
								expectedInstanceAddress,
								models.NewPortMapping(expectedExternalPort, expectedContainerPort),
								models.NewPortMapping(expectedAdditionalExternalPort, expectedAdditionalContainerPort),
							),
							State:          models.ActualLRPStateCrashed,
							PlacementState: models.PlacementStateType_Normal,
						},
					}

					routeHandler.HandleEvent(logger, models.NewFlattenedActualLRPRemovedEvent(actualLRP))
				})

				It("should NOT log the net info", func() {
					Expect(logger).ToNot(gbytes.Say(
						fmt.Sprintf(
							`"net_info":\{"address":"%s","ports":\[\{"container_port":%d,"host_port":%d\},\{"container_port":%d,"host_port":%d\}\],"instance_address":"%s"\}`,
							expectedHost,
							expectedContainerPort,
							expectedExternalPort,
							expectedAdditionalContainerPort,
							expectedExternalPort,
							expectedInstanceAddress,
						),
					))
				})

				It("doesn't remove the endpoint from the table", func() {
					Expect(fakeTable.RemoveEndpointCallCount()).To(Equal(0))
				})

				It("doesn't emit", func() {
					Expect(natsEmitter.EmitCallCount()).To(Equal(0))
				})
			})
		})
	})

	Describe("Sync", func() {
		Context("when bbs server returns desired and actual lrps", func() {
			var (
				desiredInfo []*models.DesiredLRPSchedulingInfo
				actualLRPs  []*models.FlattenedActualLRP
				domains     models.DomainSet

				endpoint1, endpoint2, endpoint3, endpoint4 routingtable.Endpoint
			)

			BeforeEach(func() {
				currentTag := &models.ModificationTag{Epoch: "abc", Index: 1}
				hostname1 := "foo.example.com"
				hostname2 := "bar.example.com"
				hostname3 := "baz.example.com"

				endpoint1 = routingtable.Endpoint{
					InstanceGUID:    "ig-1",
					Host:            "1.1.1.1",
					Index:           0,
					Port:            11,
					ContainerPort:   8080,
					Evacuating:      false,
					ModificationTag: currentTag,
				}
				endpoint2 = routingtable.Endpoint{
					InstanceGUID:    "ig-2",
					Host:            "2.2.2.2",
					Index:           0,
					Port:            22,
					ContainerPort:   8080,
					Evacuating:      false,
					ModificationTag: currentTag,
				}
				endpoint3 = routingtable.Endpoint{
					InstanceGUID:    "ig-3",
					Host:            "2.2.2.2",
					Index:           1,
					Port:            23,
					ContainerPort:   8080,
					Evacuating:      false,
					ModificationTag: currentTag,
				}

				schedulingInfo1 := &models.DesiredLRPSchedulingInfo{
					DesiredLRPKey: models.NewDesiredLRPKey("pg-1", "tests", "lg1"),
					Routes: cfroutes.CFRoutes{
						cfroutes.CFRoute{
							Hostnames:       []string{hostname1},
							Port:            8080,
							RouteServiceUrl: "https://rs.example.com",
						},
					}.RoutingInfo(),
					Instances: 1,
				}

				schedulingInfo2 := &models.DesiredLRPSchedulingInfo{
					DesiredLRPKey: models.NewDesiredLRPKey("pg-2", "tests", "lg2"),
					Routes: cfroutes.CFRoutes{
						cfroutes.CFRoute{
							Hostnames: []string{hostname2},
							Port:      8080,
						},
					}.RoutingInfo(),
					Instances: 1,
				}

				schedulingInfo3 := &models.DesiredLRPSchedulingInfo{
					DesiredLRPKey: models.NewDesiredLRPKey("pg-3", "tests", "lg3"),
					Routes: cfroutes.CFRoutes{
						cfroutes.CFRoute{
							Hostnames: []string{hostname3},
							Port:      8080,
						},
					}.RoutingInfo(),
					Instances: 2,
				}

				actualLRP1 := &models.FlattenedActualLRP{
					ActualLRPKey:         models.NewActualLRPKey("pg-1", 0, "domain"),
					ActualLRPInstanceKey: models.NewActualLRPInstanceKey(endpoint1.InstanceGUID, "cell-id"),
					ActualLRPInfo: models.ActualLRPInfo{
						ActualLRPNetInfo: models.NewActualLRPNetInfo(endpoint1.Host, "container-ip-1", models.NewPortMapping(endpoint1.Port, endpoint1.ContainerPort)),
						State:            models.ActualLRPStateRunning,
						PlacementState:   models.PlacementStateType_Normal,
					},
				}

				actualLRP2 := &models.FlattenedActualLRP{
					ActualLRPKey:         models.NewActualLRPKey("pg-2", 0, "domain"),
					ActualLRPInstanceKey: models.NewActualLRPInstanceKey(endpoint2.InstanceGUID, "cell-id"),
					ActualLRPInfo: models.ActualLRPInfo{
						ActualLRPNetInfo: models.NewActualLRPNetInfo(endpoint2.Host, "container-ip-2", models.NewPortMapping(endpoint2.Port, endpoint2.ContainerPort)),
						State:            models.ActualLRPStateRunning,
						PlacementState:   models.PlacementStateType_Normal,
					},
				}

				actualLRP3 := &models.FlattenedActualLRP{
					ActualLRPKey:         models.NewActualLRPKey("pg-3", 1, "domain"),
					ActualLRPInstanceKey: models.NewActualLRPInstanceKey(endpoint3.InstanceGUID, "cell-id"),
					ActualLRPInfo: models.ActualLRPInfo{
						ActualLRPNetInfo: models.NewActualLRPNetInfo(endpoint3.Host, "container-ip-3", models.NewPortMapping(endpoint3.Port, endpoint3.ContainerPort)),
						State:            models.ActualLRPStateRunning,
						PlacementState:   models.PlacementStateType_Normal,
					},
				}

				desiredInfo = []*models.DesiredLRPSchedulingInfo{
					schedulingInfo1, schedulingInfo2, schedulingInfo3,
				}
				actualLRPs = []*models.FlattenedActualLRP{
					actualLRP1,
					actualLRP2,
					actualLRP3,
				}

				domains = models.NewDomainSet([]string{"domain"})

				routesByRoutingKey := func(schedulingInfos []*models.DesiredLRPSchedulingInfo) map[routingtable.RoutingKey][]routingtable.Route {
					byRoutingKey := map[routingtable.RoutingKey][]routingtable.Route{}
					for _, desired := range schedulingInfos {
						routes, err := cfroutes.CFRoutesFromRoutingInfo(desired.Routes)
						if err == nil && len(routes) > 0 {
							for _, cfRoute := range routes {
								key := routingtable.RoutingKey{ProcessGUID: desired.ProcessGuid, ContainerPort: cfRoute.Port}
								var routeEntries []routingtable.Route
								for _, hostname := range cfRoute.Hostnames {
									routeEntries = append(routeEntries, routingtable.Route{
										Hostname:         hostname,
										LogGUID:          desired.LogGuid,
										RouteServiceUrl:  cfRoute.RouteServiceUrl,
										IsolationSegment: cfRoute.IsolationSegment,
									})
								}
								byRoutingKey[key] = append(byRoutingKey[key], routeEntries...)
							}
						}
					}

					return byRoutingKey
				}

				fakeTable.SwapStub = func(t routingtable.RoutingTable, d models.DomainSet) (routingtable.TCPRouteMappings, routingtable.MessagesToEmit) {

					routes := routesByRoutingKey(desiredInfo)
					routesList := make([]routingtable.Route, 3)
					for _, route := range routes {
						routesList = append(routesList, route[0])
					}

					return emptyTCPRouteMappings, routingtable.MessagesToEmit{
						RegistrationMessages: []routingtable.RegistryMessage{
							routingtable.RegistryMessageFor(endpoint1, routesList[0], true),
							routingtable.RegistryMessageFor(endpoint2, routesList[1], true),
							routingtable.RegistryMessageFor(endpoint3, routesList[2], true),
						},
					}
				}
			})

			It("updates the routing table", func() {
				routeHandler.Sync(logger, desiredInfo, actualLRPs, domains, nil)
				Expect(fakeTable.SwapCallCount()).Should(Equal(1))
				tempRoutingTable, swapDomains := fakeTable.SwapArgsForCall(0)
				Expect(tempRoutingTable.HTTPAssociationsCount()).To(Equal(3))
				Expect(swapDomains).To(Equal(domains))

				Expect(natsEmitter.EmitCallCount()).Should(Equal(1))
			})

			Context("when emitting metrics in localMode", func() {
				BeforeEach(func() {
					routeHandler = routehandlers.NewHandler(fakeTable, natsEmitter, nil, true, fakeMetronClient)
					fakeTable.HTTPAssociationsCountReturns(5)
				})

				It("emits the HTTPRouteCount", func() {
					routeHandler.Sync(logger, desiredInfo, actualLRPs, domains, nil)
					Eventually(metricChan).Should(Receive(Equal(metric{
						name:  "HTTPRouteCount",
						value: 5,
					})))
				})
			})

			Context("when NATS events are cached", func() {
				BeforeEach(func() {
					routes := cfroutes.CFRoutes{
						cfroutes.CFRoute{
							Hostnames: []string{"anungunrama.example.com"},
							Port:      8080,
						},
					}.RoutingInfo()
					desiredLRPEvent := models.NewDesiredLRPCreatedEvent(&models.DesiredLRP{
						ProcessGuid: "pg-4",
						Routes:      &routes,
						Instances:   1,
					})

					endpoint4 = routingtable.Endpoint{
						InstanceGUID:    "ig-4",
						Host:            "3.3.3.3",
						Index:           1,
						Port:            23,
						ContainerPort:   8080,
						Evacuating:      false,
						ModificationTag: &models.ModificationTag{Epoch: "abc", Index: 1},
					}

					actualLRPEvent := models.NewFlattenedActualLRPCreatedEvent(&models.FlattenedActualLRP{
						ActualLRPKey:         models.NewActualLRPKey("pg-4", 0, "domain"),
						ActualLRPInstanceKey: models.NewActualLRPInstanceKey(endpoint4.InstanceGUID, "cell-id"),
						ActualLRPInfo: models.ActualLRPInfo{
							ActualLRPNetInfo: models.NewActualLRPNetInfo(endpoint4.Host, "container-ip-4", models.NewPortMapping(endpoint4.Port, endpoint4.ContainerPort)),
							State:            models.ActualLRPStateRunning,
							PlacementState:   models.PlacementStateType_Normal,
						},
					})

					cachedEvents := map[string]models.Event{
						desiredLRPEvent.Key(): desiredLRPEvent,
						actualLRPEvent.Key():  actualLRPEvent,
					}
					routeHandler.Sync(
						logger,
						desiredInfo,
						actualLRPs,
						domains,
						cachedEvents,
					)
				})

				It("updates the routing table and emit cached events", func() {
					Expect(fakeTable.SwapCallCount()).Should(Equal(1))
					tempRoutingTable, _ := fakeTable.SwapArgsForCall(0)
					Expect(tempRoutingTable.HTTPAssociationsCount()).Should(Equal(4))
					Expect(natsEmitter.EmitCallCount()).Should(Equal(1))
				})
			})
		})
	})

	Describe("EmitExternal", func() {
		var registrationMsgs routingtable.MessagesToEmit
		BeforeEach(func() {
			currentTag := &models.ModificationTag{Epoch: "abc", Index: 1}
			endpoint1 := routingtable.Endpoint{
				InstanceGUID:    "ig-1",
				Host:            "1.1.1.1",
				Index:           0,
				Port:            11,
				ContainerPort:   8080,
				Evacuating:      false,
				ModificationTag: currentTag,
			}
			endpoint2 := routingtable.Endpoint{
				InstanceGUID:    "ig-2",
				Host:            "2.2.2.2",
				Index:           0,
				Port:            22,
				ContainerPort:   8080,
				Evacuating:      false,
				ModificationTag: currentTag,
			}
			endpoint3 := routingtable.Endpoint{
				InstanceGUID:    "ig-3",
				Host:            "2.2.2.2",
				Index:           1,
				Port:            23,
				ContainerPort:   8080,
				Evacuating:      false,
				ModificationTag: currentTag,
			}
			route := routingtable.Route{}
			registrationMsgs = routingtable.MessagesToEmit{
				RegistrationMessages: []routingtable.RegistryMessage{
					routingtable.RegistryMessageFor(endpoint1, route, true),
					routingtable.RegistryMessageFor(endpoint2, route, true),
					routingtable.RegistryMessageFor(endpoint3, route, true),
				},
			}

			fakeTable.GetExternalRoutingEventsReturns(emptyTCPRouteMappings, registrationMsgs)
			fakeTable.HTTPAssociationsCountReturns(3)
		})
		It("emits all registration events", func() {
			routeHandler.EmitExternal(logger)
			Expect(fakeTable.GetExternalRoutingEventsCallCount()).To(Equal(1))
			Expect(natsEmitter.EmitCallCount()).To(Equal(1))
			Expect(natsEmitter.EmitArgsForCall(0)).To(Equal(registrationMsgs))
		})

		It("sends a 'routes total' metric", func() {
			routeHandler.EmitExternal(logger)
			Eventually(metricChan).Should(Receive(Equal(metric{
				name:  "RoutesTotal",
				value: 3,
			})))
		})

		It("sends a 'synced routes' metric", func() {
			routeHandler.EmitExternal(logger)
			Eventually(counterChan).Should(Receive(Equal(counter{
				name:  "RoutesSynced",
				delta: 3,
			})))
		})
	})

	Describe("EmitInternal", func() {
		var registrationMsgs routingtable.MessagesToEmit
		BeforeEach(func() {
			endpoint1 := routingtable.Endpoint{
				ContainerIP: "1.2.3.4",
				Since:       1,
			}
			endpoint2 := routingtable.Endpoint{
				ContainerIP: "1.2.3.5",
				Since:       2,
			}
			endpoint3 := routingtable.Endpoint{
				ContainerIP: "1.2.3.6",
				Since:       3,
			}
			registrationMsgs = routingtable.MessagesToEmit{
				InternalRegistrationMessages: []routingtable.RegistryMessage{
					{
						URIs:                 []string{"internal", "0.internal"},
						Host:                 endpoint1.ContainerIP,
						Tags:                 map[string]string{"component": "route-emitter"},
						App:                  logGuid,
						EndpointUpdatedAtNs:  endpoint1.Since,
						PrivateInstanceIndex: "0",
					},
					{
						URIs:                 []string{"internal", "0.internal"},
						Host:                 endpoint2.ContainerIP,
						Tags:                 map[string]string{"component": "route-emitter"},
						App:                  logGuid,
						EndpointUpdatedAtNs:  endpoint2.Since,
						PrivateInstanceIndex: "0",
					},
					{
						URIs:                 []string{"internal", "0.internal"},
						Host:                 endpoint3.ContainerIP,
						Tags:                 map[string]string{"component": "route-emitter"},
						App:                  logGuid,
						EndpointUpdatedAtNs:  endpoint3.Since,
						PrivateInstanceIndex: "0",
					},
				},
			}

			fakeTable.GetInternalRoutingEventsReturns(emptyTCPRouteMappings, registrationMsgs)
			fakeTable.HTTPAssociationsCountReturns(3)
		})

		It("emits all internal registration events", func() {
			routeHandler.EmitInternal(logger)
			Expect(fakeTable.GetInternalRoutingEventsCallCount()).To(Equal(1))
			Expect(natsEmitter.EmitCallCount()).To(Equal(1))
			Expect(natsEmitter.EmitArgsForCall(0)).To(Equal(registrationMsgs))
		})
	})

	Describe("RefreshDesired", func() {
		BeforeEach(func() {
			fakeTable.SetRoutesReturns(emptyTCPRouteMappings, routingtable.MessagesToEmit{})
		})

		It("adds the desired info to the routing table", func() {
			desiredInfo := &models.DesiredLRPSchedulingInfo{
				DesiredLRPKey: models.NewDesiredLRPKey("pg-1", "tests", "lg1"),
				Routes: cfroutes.CFRoutes{
					cfroutes.CFRoute{
						Hostnames:       []string{"foo.example.com"},
						Port:            8080,
						RouteServiceUrl: "https://rs.example.com",
					},
				}.RoutingInfo(),
				Instances: 1,
			}
			routeHandler.RefreshDesired(logger, []*models.DesiredLRPSchedulingInfo{desiredInfo})

			Expect(fakeTable.SetRoutesCallCount()).To(Equal(1))
			before, after := fakeTable.SetRoutesArgsForCall(0)
			Expect(before).To(BeNil())
			Expect(after).To(Equal(desiredInfo))
			Expect(natsEmitter.EmitCallCount()).Should(Equal(1))
		})
	})

	Describe("ShouldRefreshDesired", func() {
		var (
			actualLRP *models.FlattenedActualLRP
		)
		BeforeEach(func() {
			currentTag := models.ModificationTag{Epoch: "abc", Index: 1}
			endpoint1 := routingtable.Endpoint{
				InstanceGUID:    "ig-1",
				Host:            "1.1.1.1",
				Index:           0,
				Port:            11,
				ContainerPort:   8080,
				Evacuating:      false,
				ModificationTag: &currentTag,
			}

			actualLRP = &models.FlattenedActualLRP{
				ActualLRPKey:         models.NewActualLRPKey("pg-1", 0, "domain"),
				ActualLRPInstanceKey: models.NewActualLRPInstanceKey(endpoint1.InstanceGUID, "cell-id"),
				ActualLRPInfo: models.ActualLRPInfo{
					ActualLRPNetInfo: models.NewActualLRPNetInfo(endpoint1.Host,
						"container-ip-1",
						models.NewPortMapping(endpoint1.Port, endpoint1.ContainerPort),
						models.NewPortMapping(12, endpoint1.ContainerPort+1),
					),

					State:           models.ActualLRPStateRunning,
					ModificationTag: currentTag,
					PlacementState:  models.PlacementStateType_Normal,
				},
			}
		})

		Context("when corresponding desired state exists in the table", func() {
			BeforeEach(func() {
				fakeTable.HasExternalRoutesReturns(false)
			})

			It("returns false", func() {
				Expect(routeHandler.ShouldRefreshDesired(actualLRP)).To(BeTrue())
			})
		})

		Context("when corresponding desired state does not exist in the table", func() {
			BeforeEach(func() {
				fakeTable.HasExternalRoutesReturns(true)
			})

			It("returns true", func() {
				Expect(routeHandler.ShouldRefreshDesired(actualLRP)).To(BeFalse())
			})
		})
	})
})
