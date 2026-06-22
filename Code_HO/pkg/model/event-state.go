package model

type StateType string
type EventType string

const (
	EntryEvent EventType = "Entry Event"
	ExitEvent  EventType = "Exit Event"
)

// State
const (
	NULL StateType = "NULL State"
	IDLE StateType = "Idle State"

	// UE State 5GMM
	Deregistered            StateType = "Deregisterd State"
	DeregistrationInitiated StateType = "DeregistrationInitiated State"
	AuthenticationInitiated StateType = "AuthenticationInitiated State"
	RegisteredInitiated     StateType = "RegisteredInitiated State"
	Registered              StateType = "Registered State"

	// UE State 5GSM
	PDUSessionInactive        StateType = "PDUSessionInactive State"
	PDUSessionActivePending   StateType = "PDUSessionActivePending State"
	PDUSessionInactivePending StateType = "PDUSessionInactivePending State"
	PDUSessionActive          StateType = "PDUSessionActive State"
	PDUModificationPending    StateType = "PDUModificationPending State"

	// Monitor/HO State
	MonitorNull      StateType = "MonitorNull State"
	MonitorPrepare   StateType = "MonitorPrepare State"
	MonitorExecute   StateType = "MonitorExecute State"
	MonitorComplete  StateType = "MonitorComplete State"
	MonitorHoSuccess StateType = "MonitorHoSuccess State"
	MonitorHoFail    StateType = "MonitorHoFail State"

	// UE HO State
	HoStart   StateType = "HoStart State"
	HoSuccess StateType = "HoSuccess State"
	HoFail    StateType = "HoFail State"

	// Handover States
	HO_NULL        StateType = "HO_NULL State"
	HO_PREPARATION StateType = "HO_PREPARATION State"
	HO_EXECUTION   StateType = "HO_EXECUTION State"
	HO_COMPLETION  StateType = "HO_COMPLETION State"
)

// Event
const (
	Enable EventType = "Enable Event"

	// 5GMM Event
	GmmMessageEvent                   EventType = "GmmMessageEvent Event"
	InitRegistrationRequestEvent      EventType = "InitRegistrationRequestEvent Event"
	RegistrationRejectEvent           EventType = "RegistrationRejectEvent Event"
	AuthFailEvent                     EventType = "AuthFailEvent Event"
	SecurityModeFailEvent             EventType = "SecurityModeFailEvent Event"
	RegistrationAcceptEvent           EventType = "RegistrationAcceptEvent Event"
	InitDeregistrationRequestEvent    EventType = "InitDeregistrationRequestEvent Event"
	DeregistrationAcceptEvent         EventType = "DeregistrationAcceptEvent Event"
	NetworkDeregistrationRequestEvent EventType = "NetworkDeregistrationRequestEvent Event"

	MissingInfo EventType = "MissingInfo Event"

	// timer event
	T3502Event EventType = "t3502Event Event"
	T3510Event EventType = "t3510Event Event"
	T3511Event EventType = "t3511Event Event"

	// Xn handover timer events
	TXnRELOCprepExpiredEvent    EventType = "TXnRELOCprepExpiredEvent Event"
	TXnRELOCoverallExpiredEvent EventType = "TXnRELOCoverallExpiredEvent Event"

	// N2 handover timer events
	TNGRELOCprepExpiredEvent    EventType = "TNGRELOCprepExpiredEvent Event"
	TNGRELOCoverallExpiredEvent EventType = "TNGRELOCoverallExpiredEvent Event"

	// RRC timer events
	T301ExpiredEvent EventType = "T301ExpiredEvent Event"
	T304ExpiredEvent EventType = "T304ExpiredEvent Event"
	T310ExpiredEvent EventType = "T310ExpiredEvent Event"
	T311ExpiredEvent EventType = "T311ExpiredEvent Event"
	T312ExpiredEvent EventType = "T312ExpiredEvent Event"
	T430ExpiredEvent EventType = "T430ExpiredEvent Event"

	// 5GSM Event
	InitPduSessionEstablishmentRequestEvent EventType = "InitPduSessionEstablishmentRequestEvent Event"
	EstablishmentReject                     EventType = "EstablishmentReject Event"
	EstablishmentAccept                     EventType = "EstablishmentAccept Event"
	ReleaseRequest                          EventType = "ReleaseRequest Event"
	ReleaseCommand                          EventType = "ReleaseCommand Event"
	ModificationRequest                     EventType = "ModificationRequest Event"
	ModificationCommand                     EventType = "ModificationCommand Event"
	ModificationReject                      EventType = "ModificationReject Event"
	ModificationComplete                    EventType = "ModificationComplete Event"

	// trigger command - remote event
	NullInit           EventType = "NullInit Event"
	IdleInit           EventType = "IdleInit Event"
	RegisterInit       EventType = "RegisterInit Event"
	DeregistraterInit  EventType = "DeregistraterInit Event"
	ServiceRequestInit EventType = "ServiceRequestInit Event"
	PduSessionInit     EventType = "PduSessionInit Event"
	DestroyPduSession  EventType = "DestroyPduSession Event"
	XnHandover         EventType = "XnHandover Event"
	N2Handover         EventType = "N2Handover Event"
	Terminate          EventType = "ue Event" // kill ue
	Kill               EventType = "ue Event" // force kill ue

	// Monitor/HO Event
	HoDecisionEvent            EventType = "HoDecisionEvent"
	HoXnForwardUeContextEvent  EventType = "HoXnForwardUeContextEvent"
	HoFailEvent                EventType = "HoFailEvent"
	HoRlinkPrepareReqSentEvent EventType = "HoRlinkPrepareReqSentEvent"
	HoPathSwitchRequestEvent   EventType = "HoPathSwitchRequestEvent"
	HoPathSwitchFailEvent      EventType = "HoPathSwitchFailEvent"
	HoRlinkSetupPduSessonEvent EventType = "HoRlinkSetupPduSessonEvent"
	HoUeConnectedEvent         EventType = "HoUeConnectedEvent"
	HoPathSwitchReqEvent       EventType = "HoPathSwitchReqEvent"
	HoPathSwitchAckEvent       EventType = "HoPathSwitchAckEvent"
	HoRlinkPrepareRespEvent    EventType = "HoRlinkPrepareRespEvent"

	// Handover Events
	HO_PrepareEvent  EventType = "HO_PrepareEvent Event"
	HO_ExecuteEvent  EventType = "HO_ExecuteEvent Event"
	HO_CompleteEvent EventType = "HO_CompleteEvent Event"
	HandoverFinished EventType = "HandoverFinished Event"

	// CHO (Conditional Handover) Events
	MeasurementReportEvent   EventType = "MeasurementReportEvent Event"
	ChoConfigEvent           EventType = "ChoConfigEvent Event"
	ChoConditionMetEvent     EventType = "ChoConditionMetEvent Event"
	RRCReconfigCompleteEvent EventType = "RRCReconfigCompleteEvent Event"
	ChoCancelEvent           EventType = "ChoCancelEvent Event"
)
