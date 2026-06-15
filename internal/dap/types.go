package dap

import (
	"encoding/json"
	"time"
)

// ---------------------------------------------------------------------------
// Constants
// ---------------------------------------------------------------------------

const (
	DefaultRequestTimeout = 30 * time.Second
	MaxOutputBytes        = 128 * 1024
	IdleTimeout           = 10 * time.Minute
	HeartbeatInterval     = 5 * time.Second
)

// ---------------------------------------------------------------------------
// Session status
// ---------------------------------------------------------------------------

// DapSessionStatus describes the lifecycle state of a debug session.
type DapSessionStatus string

const (
	DapStatusLaunching   DapSessionStatus = "launching"
	DapStatusConfiguring DapSessionStatus = "configuring"
	DapStatusStopped     DapSessionStatus = "stopped"
	DapStatusRunning     DapSessionStatus = "running"
	DapStatusTerminated  DapSessionStatus = "terminated"
)

// ---------------------------------------------------------------------------
// Event types
// ---------------------------------------------------------------------------

// DapEventType categorises the kind of DAP event.
type DapEventType string

const (
	DapEventStopped     DapEventType = "stopped"
	DapEventContinued   DapEventType = "continued"
	DapEventOutput      DapEventType = "output"
	DapEventExited      DapEventType = "exited"
	DapEventTerminated  DapEventType = "terminated"
	DapEventInitialized DapEventType = "initialized"
)

// DapEventHandler is a callback that receives a DAP event body together with
// the raw event message.
type DapEventHandler func(body interface{}, event *DapEventMessage)

// ---------------------------------------------------------------------------
// Protocol message base
// ---------------------------------------------------------------------------

// DapProtocolMessage is the wire‑level envelope shared by every DAP message.
type DapProtocolMessage struct {
	Seq  int    `json:"seq"`
	Type string `json:"type"`
}

// DapRequestMessage is a request sent from the client to the debug adapter.
type DapRequestMessage struct {
	DapProtocolMessage

	Command   string          `json:"command"`
	Arguments json.RawMessage `json:"arguments,omitempty"`
}

// DapResponseMessage is a response sent from the debug adapter back to the
// client.  The Success field indicates whether the corresponding request was
// handled successfully.
type DapResponseMessage struct {
	DapProtocolMessage

	RequestSeq int             `json:"request_seq"`
	Success    bool            `json:"success"`
	Command    string          `json:"command"`
	Message    string          `json:"message,omitempty"`
	Body       json.RawMessage `json:"body,omitempty"`
}

// DapEventMessage is an event pushed from the debug adapter to the client.
type DapEventMessage struct {
	DapProtocolMessage

	Event string          `json:"event"`
	Body  json.RawMessage `json:"body,omitempty"`
}

// ---------------------------------------------------------------------------
// Initialize
// ---------------------------------------------------------------------------

// DapInitializeArguments is the arguments for the "initialize" request sent by
// the client to the adapter.
type DapInitializeArguments struct {
	ClientID                            string `json:"clientID,omitempty"`
	ClientName                          string `json:"clientName,omitempty"`
	AdapterID                           string `json:"adapterID"`
	PathFormat                          string `json:"pathFormat,omitempty"`
	LinesStartAt1                       bool   `json:"linesStartAt1,omitempty"`
	ColumnsStartAt1                     bool   `json:"columnsStartAt1,omitempty"`
	Locale                              string `json:"locale,omitempty"`
	SupportsVariableType                bool   `json:"supportsVariableType,omitempty"`
	SupportsVariablePaging              bool   `json:"supportsVariablePaging,omitempty"`
	SupportsRunInTerminalRequest        bool   `json:"supportsRunInTerminalRequest,omitempty"`
	SupportsProgressReporting           bool   `json:"supportsProgressReporting,omitempty"`
	SupportsInvalidatedEvent            bool   `json:"supportsInvalidatedEvent,omitempty"`
	SupportsMemoryReferences            bool   `json:"supportsMemoryReferences,omitempty"`
	SupportsArgsCanBeInterpretedByShell bool   `json:"supportsArgsCanBeInterpretedByShell,omitempty"`
	SupportsStartDebuggingRequest       bool   `json:"supportsStartDebuggingRequest,omitempty"`
}

// DapCapabilities is the response body for the "initialize" request.  It
// describes the capabilities of the debug adapter.
type DapCapabilities struct {
	SupportsConfigurationDoneRequest      bool `json:"supportsConfigurationDoneRequest,omitempty"`
	SupportsFunctionBreakpoints           bool `json:"supportsFunctionBreakpoints,omitempty"`
	SupportsConditionalBreakpoints        bool `json:"supportsConditionalBreakpoints,omitempty"`
	SupportsHitConditionalBreakpoints     bool `json:"supportsHitConditionalBreakpoints,omitempty"`
	SupportsEvaluateForHits               bool `json:"supportsEvaluateForHits,omitempty"`
	SupportsStepBack                      bool `json:"supportsStepBack,omitempty"`
	SupportsSetVariable                   bool `json:"supportsSetVariable,omitempty"`
	SupportsRestartFrame                  bool `json:"supportsRestartFrame,omitempty"`
	SupportsGotoTargetsRequest            bool `json:"supportsGotoTargetsRequest,omitempty"`
	SupportsStepInTargetsRequest          bool `json:"supportsStepInTargetsRequest,omitempty"`
	SupportsCompletionsRequest            bool `json:"supportsCompletionsRequest,omitempty"`
	SupportsModulesRequest                bool `json:"supportsModulesRequest,omitempty"`
	SupportsRestartRequest                bool `json:"supportsRestartRequest,omitempty"`
	SupportsExceptionOptions              bool `json:"supportsExceptionOptions,omitempty"`
	SupportsValueFormattingOptions        bool `json:"supportsValueFormattingOptions,omitempty"`
	SupportsExceptionInfoRequest          bool `json:"supportsExceptionInfoRequest,omitempty"`
	SupportTerminateDebuggee              bool `json:"supportTerminateDebuggee,omitempty"`
	SupportsSuspendDebuggee               bool `json:"supportsSuspendDebuggee,omitempty"`
	SupportsDelayedStackTraceLoading      bool `json:"supportsDelayedStackTraceLoading,omitempty"`
	SupportsLoadedSourcesRequest          bool `json:"supportsLoadedSourcesRequest,omitempty"`
	SupportsLogPoints                     bool `json:"supportsLogPoints,omitempty"`
	SupportsTerminateThreadsRequest       bool `json:"supportsTerminateThreadsRequest,omitempty"`
	SupportsSetExpression                 bool `json:"supportsSetExpression,omitempty"`
	SupportsMemoryReferences              bool `json:"supportsMemoryReferences,omitempty"`
	SupportsReadMemoryRequest             bool `json:"supportsReadMemoryRequest,omitempty"`
	SupportsWriteMemoryRequest            bool `json:"supportsWriteMemoryRequest,omitempty"`
	SupportsDisassembleRequest            bool `json:"supportsDisassembleRequest,omitempty"`
	SupportsCancelRequest                 bool `json:"supportsCancelRequest,omitempty"`
	SupportsBreakpointLocationsRequest    bool `json:"supportsBreakpointLocationsRequest,omitempty"`
	SupportsClipboardContext              bool `json:"supportsClipboardContext,omitempty"`
	SupportsSteppingGranularity           bool `json:"supportsSteppingGranularity,omitempty"`
	SupportsInstructionBreakpoints        bool `json:"supportsInstructionBreakpoints,omitempty"`
	SupportsExceptionFilterOptions        bool `json:"supportsExceptionFilterOptions,omitempty"`
	SupportsSingleThreadExecutionRequests bool `json:"supportsSingleThreadExecutionRequests,omitempty"`
}

// ---------------------------------------------------------------------------
// Launch / Attach
// ---------------------------------------------------------------------------

// DapLaunchArguments is the arguments for the "launch" request.
type DapLaunchArguments struct {
	Program     string   `json:"program,omitempty"`
	Args        []string `json:"args,omitempty"`
	Cwd         string   `json:"cwd,omitempty"`
	StopOnEntry bool     `json:"stopOnEntry,omitempty"`
}

// DapAttachArguments is the arguments for the "attach" request.
type DapAttachArguments struct {
	PID       int    `json:"pid,omitempty"`
	ProcessID int    `json:"processId,omitempty"`
	Port      int    `json:"port,omitempty"`
	Host      string `json:"host,omitempty"`
	Cwd       string `json:"cwd,omitempty"`
}

// ---------------------------------------------------------------------------
// Source & breakpoints
// ---------------------------------------------------------------------------

// DapSource identifies a source file or a source reference.
type DapSource struct {
	Name             string          `json:"name,omitempty"`
	Path             string          `json:"path,omitempty"`
	SourceReference  int             `json:"sourceReference,omitempty"`
	PresentationHint string          `json:"presentationHint,omitempty"`
	Origin           string          `json:"origin,omitempty"`
	Sources          []*DapSource    `json:"sources,omitempty"`
	AdapterData      json.RawMessage `json:"adapterData,omitempty"`
}

// DapSourceBreakpoint describes a breakpoint to be set on a source line.
type DapSourceBreakpoint struct {
	Line         int    `json:"line"`
	Column       int    `json:"column,omitempty"`
	Condition    string `json:"condition,omitempty"`
	HitCondition string `json:"hitCondition,omitempty"`
	LogMessage   string `json:"logMessage,omitempty"`
}

// DapBreakpoint represents an installed (or pending) breakpoint returned by
// the adapter.
type DapBreakpoint struct {
	Id                   int        `json:"id,omitempty"`
	Verified             bool       `json:"verified"`
	Message              string     `json:"message,omitempty"`
	Source               *DapSource `json:"source,omitempty"`
	Line                 int        `json:"line,omitempty"`
	Column               int        `json:"column,omitempty"`
	EndLine              int        `json:"endLine,omitempty"`
	EndColumn            int        `json:"endColumn,omitempty"`
	InstructionReference string     `json:"instructionReference,omitempty"`
	Offset               int        `json:"offset,omitempty"`
}

// DapSetBreakpointsArguments is the arguments for the "setBreakpoints"
// request.
type DapSetBreakpointsArguments struct {
	Source         *DapSource            `json:"source"`
	Breakpoints    []DapSourceBreakpoint `json:"breakpoints,omitempty"`
	Lines          []int                 `json:"lines,omitempty"`
	SourceModified bool                  `json:"sourceModified,omitempty"`
}

// DapSetBreakpointsResponse is the body of the "setBreakpoints" response.
type DapSetBreakpointsResponse struct {
	Breakpoints []DapBreakpoint `json:"breakpoints"`
}

// DapFunctionBreakpoint describes a breakpoint to be set on a function.
type DapFunctionBreakpoint struct {
	Name         string `json:"name"`
	Condition    string `json:"condition,omitempty"`
	HitCondition string `json:"hitCondition,omitempty"`
}

// DapSetFunctionBreakpointsArguments is the arguments for the
// "setFunctionBreakpoints" request.
type DapSetFunctionBreakpointsArguments struct {
	Breakpoints []DapFunctionBreakpoint `json:"breakpoints"`
}

// DapInstructionBreakpoint describes a breakpoint to be set at an instruction.
type DapInstructionBreakpoint struct {
	InstructionReference string `json:"instructionReference"`
	Offset               int    `json:"offset,omitempty"`
	Condition            string `json:"condition,omitempty"`
	HitCondition         string `json:"hitCondition,omitempty"`
}

// DapSetInstructionBreakpointsArguments is the arguments for the
// "setInstructionBreakpoints" request.
type DapSetInstructionBreakpointsArguments struct {
	Breakpoints []DapInstructionBreakpoint `json:"breakpoints"`
}

// DapDataBreakpointInfoArguments is the arguments for the
// "dataBreakpointInfo" request.
type DapDataBreakpointInfoArguments struct {
	VariableReference int    `json:"variableReference"`
	Name              string `json:"name"`
}

// DapDataBreakpointInfoResponse is the body of the "dataBreakpointInfo"
// response.
type DapDataBreakpointInfoResponse struct {
	DataId      string   `json:"dataId"`
	Description string   `json:"description"`
	AccessTypes []string `json:"accessTypes,omitempty"`
	CanPersist  bool     `json:"canPersist,omitempty"`
}

// DapDataBreakpoint describes a data breakpoint.
type DapDataBreakpoint struct {
	DataId       string `json:"dataId"`
	AccessType   string `json:"accessType,omitempty"`
	Condition    string `json:"condition,omitempty"`
	HitCondition string `json:"hitCondition,omitempty"`
}

// DapSetDataBreakpointsArguments is the arguments for the
// "setDataBreakpoints" request.
type DapSetDataBreakpointsArguments struct {
	Breakpoints []DapDataBreakpoint `json:"breakpoints"`
}

// ---------------------------------------------------------------------------
// Execution control
// ---------------------------------------------------------------------------

// DapContinueArguments is the arguments for the "continue" request.
type DapContinueArguments struct {
	ThreadId int `json:"threadId"`
}

// DapStepArguments is used for the "next", "stepIn", "stepOut" requests.
type DapStepArguments struct {
	ThreadId    int    `json:"threadId"`
	Granularity string `json:"granularity,omitempty"`
}

// DapPauseArguments is the arguments for the "pause" request.
type DapPauseArguments struct {
	ThreadId int `json:"threadId"`
}

// ---------------------------------------------------------------------------
// Stack trace
// ---------------------------------------------------------------------------

// DapStackTraceArguments is the arguments for the "stackTrace" request.
type DapStackTraceArguments struct {
	ThreadId   int    `json:"threadId"`
	StartFrame int    `json:"startFrame,omitempty"`
	Levels     int    `json:"levels,omitempty"`
	Format     string `json:"format,omitempty"`
}

// DapStackFrame represents a single stack frame.
type DapStackFrame struct {
	Id                          int         `json:"id"`
	Name                        string      `json:"name"`
	Source                      *DapSource  `json:"source,omitempty"`
	Line                        int         `json:"line"`
	Column                      int         `json:"column"`
	EndLine                     int         `json:"endLine,omitempty"`
	EndColumn                   int         `json:"endColumn,omitempty"`
	CanRestart                  bool        `json:"canRestart,omitempty"`
	InstructionPointerReference string      `json:"instructionPointerReference,omitempty"`
	ModuleId                    interface{} `json:"moduleId,omitempty"`
	PresentationHint            string      `json:"presentationHint,omitempty"`
}

// DapStackTraceResponse is the body of the "stackTrace" response.
type DapStackTraceResponse struct {
	StackFrames []DapStackFrame `json:"stackFrames"`
	TotalFrames int             `json:"totalFrames,omitempty"`
}

// ---------------------------------------------------------------------------
// Scopes
// ---------------------------------------------------------------------------

// DapScopesArguments is the arguments for the "scopes" request.
type DapScopesArguments struct {
	FrameId int `json:"frameId"`
}

// DapScope represents a variable scope (e.g. local, global, closure).
type DapScope struct {
	Name               string     `json:"name"`
	VariablesReference int        `json:"variablesReference"`
	NamedVariables     int        `json:"namedVariables,omitempty"`
	IndexedVariables   int        `json:"indexedVariables,omitempty"`
	Expensive          bool       `json:"expensive"`
	PresentationHint   string     `json:"presentationHint,omitempty"`
	Source             *DapSource `json:"source,omitempty"`
	Line               int        `json:"line,omitempty"`
	Column             int        `json:"column,omitempty"`
	EndLine            int        `json:"endLine,omitempty"`
	EndColumn          int        `json:"endColumn,omitempty"`
}

// DapScopesResponse is the body of the "scopes" response.
type DapScopesResponse struct {
	Scopes []DapScope `json:"scopes"`
}

// ---------------------------------------------------------------------------
// Variables
// ---------------------------------------------------------------------------

// DapVariablesArguments is the arguments for the "variables" request.
type DapVariablesArguments struct {
	VariablesReference int    `json:"variablesReference"`
	Filter             string `json:"filter,omitempty"`
	Start              int    `json:"start,omitempty"`
	Count              int    `json:"count,omitempty"`
	Format             string `json:"format,omitempty"`
}

// DapVariable represents a variable returned by the adapter.
type DapVariable struct {
	Name               string          `json:"name"`
	Value              string          `json:"value"`
	Type               string          `json:"type,omitempty"`
	PresentationHint   json.RawMessage `json:"presentationHint,omitempty"`
	EvaluateName       string          `json:"evaluateName,omitempty"`
	VariablesReference int             `json:"variablesReference"`
	NamedVariables     int             `json:"namedVariables,omitempty"`
	IndexedVariables   int             `json:"indexedVariables,omitempty"`
	MemoryReference    string          `json:"memoryReference,omitempty"`
	DeclaredType       string          `json:"declaredType,omitempty"`
}

// DapVariablesResponse is the body of the "variables" response.
type DapVariablesResponse struct {
	Variables []DapVariable `json:"variables"`
}

// ---------------------------------------------------------------------------
// Threads
// ---------------------------------------------------------------------------

// DapThread represents a single thread in the debuggee.
type DapThread struct {
	Id   int    `json:"id"`
	Name string `json:"name"`
}

// DapThreadsResponse is the body of the "threads" response.
type DapThreadsResponse struct {
	Threads []DapThread `json:"threads"`
}

// ---------------------------------------------------------------------------
// Evaluate
// ---------------------------------------------------------------------------

// DapEvaluateArguments is the arguments for the "evaluate" request.
type DapEvaluateArguments struct {
	Expression string `json:"expression"`
	FrameId    int    `json:"frameId,omitempty"`
	Context    string `json:"context,omitempty"`
	Format     string `json:"format,omitempty"`
}

// DapEvaluateResponse is the body of the "evaluate" response.
type DapEvaluateResponse struct {
	Result             string          `json:"result"`
	Type               string          `json:"type,omitempty"`
	PresentationHint   json.RawMessage `json:"presentationHint,omitempty"`
	VariablesReference int             `json:"variablesReference"`
	NamedVariables     int             `json:"namedVariables,omitempty"`
	IndexedVariables   int             `json:"indexedVariables,omitempty"`
	MemoryReference    string          `json:"memoryReference,omitempty"`
}

// ---------------------------------------------------------------------------
// Disassembly
// ---------------------------------------------------------------------------

// DapDisassembleArguments is the arguments for the "disassemble" request.
type DapDisassembleArguments struct {
	MemoryReference   string `json:"memoryReference"`
	InstructionOffset int    `json:"instructionOffset,omitempty"`
	InstructionCount  int    `json:"instructionCount"`
	SymbolOffset      int    `json:"symbolOffset,omitempty"`
	SymbolGranularity string `json:"symbolGranularity,omitempty"`
	Offset            int    `json:"offset,omitempty"`
	Granularity       string `json:"granularity,omitempty"`
	Memory            string `json:"memory,omitempty"`
}

// DapDisassembledInstruction represents a single disassembled instruction.
type DapDisassembledInstruction struct {
	Address          string `json:"address"`
	InstructionBytes string `json:"instructionBytes,omitempty"`
	Instruction      string `json:"instruction"`
	Symbol           string `json:"symbol,omitempty"`
	Offset           int    `json:"offset,omitempty"`
	Line             int    `json:"line,omitempty"`
	Column           int    `json:"column,omitempty"`
	EndLine          int    `json:"endLine,omitempty"`
	EndColumn        int    `json:"endColumn,omitempty"`
	PresentationHint string `json:"presentationHint,omitempty"`
	Location         string `json:"location,omitempty"`
}

// DapDisassembleResponse is the body of the "disassemble" response.
type DapDisassembleResponse struct {
	Instructions []DapDisassembledInstruction `json:"instructions"`
}

// ---------------------------------------------------------------------------
// Memory
// ---------------------------------------------------------------------------

// DapReadMemoryArguments is the arguments for the "readMemory" request.
type DapReadMemoryArguments struct {
	MemoryReference string `json:"memoryReference"`
	Offset          int    `json:"offset,omitempty"`
	Count           int    `json:"count"`
}

// DapReadMemoryResponse is the body of the "readMemory" response.
type DapReadMemoryResponse struct {
	Address  string `json:"address"`
	Unmapped bool   `json:"unmapped,omitempty"`
	Offset   int    `json:"offset,omitempty"`
	Data     string `json:"data,omitempty"`
}

// DapWriteMemoryArguments is the arguments for the "writeMemory" request.
type DapWriteMemoryArguments struct {
	MemoryReference string `json:"memoryReference"`
	Offset          int    `json:"offset,omitempty"`
	Data            string `json:"data"`
}

// DapWriteMemoryResponse is the body of the "writeMemory" response.
type DapWriteMemoryResponse struct {
	Offset       int `json:"offset,omitempty"`
	BytesWritten int `json:"bytesWritten,omitempty"`
}

// ---------------------------------------------------------------------------
// Modules
// ---------------------------------------------------------------------------

// DapModule describes a loaded module (e.g. a shared library).
type DapModule struct {
	Id             interface{} `json:"id"`
	Name           string      `json:"name"`
	Path           string      `json:"path,omitempty"`
	IsOptimized    bool        `json:"isOptimized,omitempty"`
	IsUserCode     bool        `json:"isUserCode,omitempty"`
	Version        string      `json:"version,omitempty"`
	SymbolStatus   string      `json:"symbolStatus,omitempty"`
	SymbolFilePath string      `json:"symbolFilePath,omitempty"`
	DateTimeStamp  string      `json:"dateTimeStamp,omitempty"`
	AddressRange   string      `json:"addressRange,omitempty"`
}

// DapModulesArguments is the arguments for the "modules" request.
type DapModulesArguments struct {
	StartModule int `json:"startModule,omitempty"`
	ModuleCount int `json:"moduleCount,omitempty"`
}

// DapModulesResponse is the body of the "modules" response.
type DapModulesResponse struct {
	Modules      []DapModule `json:"modules"`
	TotalModules int         `json:"totalModules,omitempty"`
}

// DapLoadedSourcesResponse is the body of the "loadedSources" response.
type DapLoadedSourcesResponse struct {
	Sources []DapSource `json:"sources"`
}

// ---------------------------------------------------------------------------
// Event bodies
// ---------------------------------------------------------------------------

// DapStoppedEventBody is the body of the "stopped" event.
type DapStoppedEventBody struct {
	Reason            string `json:"reason"`
	Description       string `json:"description,omitempty"`
	ThreadId          int    `json:"threadId,omitempty"`
	PreserveFocusHint bool   `json:"preserveFocusHint,omitempty"`
	Text              string `json:"text,omitempty"`
	AllThreadsStopped bool   `json:"allThreadsStopped,omitempty"`
	HitBreakpointIds  []int  `json:"hitBreakpointIds,omitempty"`
}

// DapContinuedEventBody is the body of the "continued" event.
type DapContinuedEventBody struct {
	ThreadId            int  `json:"threadId"`
	AllThreadsContinued bool `json:"allThreadsContinued,omitempty"`
}

// DapOutputEventBody is the body of the "output" event.
type DapOutputEventBody struct {
	Category     string          `json:"category,omitempty"`
	Output       string          `json:"output"`
	VariablesRef int             `json:"variablesReference,omitempty"`
	Source       *DapSource      `json:"source,omitempty"`
	Line         int             `json:"line,omitempty"`
	Column       int             `json:"column,omitempty"`
	Data         json.RawMessage `json:"data,omitempty"`
}

// DapExitedEventBody is the body of the "exited" event.
type DapExitedEventBody struct {
	ExitCode int `json:"exitCode"`
}

// DapTerminatedEventBody is the body of the "terminated" event.
type DapTerminatedEventBody struct {
	Restart bool `json:"restart,omitempty"`
}

// ---------------------------------------------------------------------------
// Internal tracking types
// ---------------------------------------------------------------------------

// DapBreakpointRecord tracks the state of a source breakpoint within the session.
type DapBreakpointRecord struct {
	ID         int                 `json:"id"`
	Breakpoint DapSourceBreakpoint `json:"breakpoint"`
	Verified   bool                `json:"verified"`
	ActualLine int                 `json:"actualLine,omitempty"`
}

// DapFunctionBreakpointRecord tracks the state of a function breakpoint.
type DapFunctionBreakpointRecord struct {
	ID         int                   `json:"id"`
	Breakpoint DapFunctionBreakpoint `json:"breakpoint"`
	Verified   bool                  `json:"verified"`
}

// DapInstructionBreakpointRecord tracks the state of an instruction breakpoint.
type DapInstructionBreakpointRecord struct {
	ID         int                      `json:"id"`
	Breakpoint DapInstructionBreakpoint `json:"breakpoint"`
	Verified   bool                     `json:"verified"`
}

// DapDataBreakpointRecord tracks the state of a data breakpoint.
type DapDataBreakpointRecord struct {
	ID         int               `json:"id"`
	Breakpoint DapDataBreakpoint `json:"breakpoint"`
	Verified   bool              `json:"verified"`
}

// DapStopLocation captures where the debuggee stopped.
type DapStopLocation struct {
	ThreadID int        `json:"threadId"`
	Reason   string     `json:"reason"`
	Source   *DapSource `json:"source,omitempty"`
	Line     int        `json:"line,omitempty"`
	Column   int        `json:"column,omitempty"`
	Text     string     `json:"text,omitempty"`
}

// DapSessionSummary holds high‑level state for a debug session.
type DapSessionSummary struct {
	ID           string           `json:"id"`
	Status       DapSessionStatus `json:"status"`
	IsLocal      bool             `json:"isLocal"`
	Adapter      string           `json:"adapter,omitempty"`
	ProcessName  string           `json:"processName,omitempty"`
	ProcessID    int              `json:"processId,omitempty"`
	StopLocation *DapStopLocation `json:"stopLocation,omitempty"`
	Threads      []DapThread      `json:"threads,omitempty"`
	HasExited    bool             `json:"hasExited,omitempty"`
}

// DapContinueOutcome is returned after a continue request completes and
// includes a snapshot of the session state.
type DapContinueOutcome struct {
	Snapshot DapSessionSummary `json:"snapshot"`
	State    string            `json:"state"`
	TimedOut bool              `json:"timedOut"`
}

// DapAdapterConfig describes how to resolve and launch a debug adapter.
type DapAdapterConfig struct {
	Type           string                 `json:"type"`
	Command        string                 `json:"command,omitempty"`
	Args           []string               `json:"args,omitempty"`
	Env            map[string]string      `json:"env,omitempty"`
	Program        string                 `json:"program,omitempty"`
	Languages      []string               `json:"languages,omitempty"`
	FileTypes      []string               `json:"fileTypes,omitempty"`
	RootMarkers    []string               `json:"rootMarkers,omitempty"`
	LaunchDefaults map[string]interface{} `json:"launchDefaults,omitempty"`
	AttachDefaults map[string]interface{} `json:"attachDefaults,omitempty"`
	ConnectMode    string                 `json:"connectMode,omitempty"`
}

// DapResolvedAdapter holds the resolved adapter configuration after
// looking up the adapter type.
type DapResolvedAdapter struct {
	Name            string                 `json:"name"`
	Command         string                 `json:"command"`
	Args            []string               `json:"args"`
	ResolvedCommand string                 `json:"resolvedCommand"`
	Languages       []string               `json:"languages,omitempty"`
	FileTypes       []string               `json:"fileTypes,omitempty"`
	RootMarkers     []string               `json:"rootMarkers,omitempty"`
	LaunchDefaults  map[string]interface{} `json:"launchDefaults,omitempty"`
	AttachDefaults  map[string]interface{} `json:"attachDefaults,omitempty"`
	ConnectMode     string                 `json:"connectMode,omitempty"`
	Env             map[string]string      `json:"env,omitempty"`
}

// DapLaunchSessionOptions configures how a launch session is started.
type DapLaunchSessionOptions struct {
	AdapterID string             `json:"adapterId"`
	Config    DapAdapterConfig   `json:"config"`
	Request   DapLaunchArguments `json:"request"`
}

// DapAttachSessionOptions configures how an attach session is started.
type DapAttachSessionOptions struct {
	AdapterID string             `json:"adapterId"`
	Config    DapAdapterConfig   `json:"config"`
	Request   DapAttachArguments `json:"request"`
}

// DapPendingRequest represents a request that has been sent to the adapter
// and is awaiting a response.
type DapPendingRequest struct {
	Command string            `json:"command"`
	Resolve func(interface{}) `json:"-"`
	Reject  func(error)       `json:"-"`
}
