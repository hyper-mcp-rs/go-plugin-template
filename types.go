package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/invopop/jsonschema"
)

// Annotations represents metadata annotations for resources and content
type Annotations struct {
	Audience     []Role     `json:"audience,omitempty"`
	LastModified *time.Time `json:"lastModified,omitempty"`
	Priority     float32    `json:"priority,omitempty"`
}

// AccessToken represents an OAuth2 access token returned by the host.
// The host handles token acquisition, caching, and refresh transparently.
type AccessToken struct {
	AccessToken string      `json:"access_token"`
	ExpiresAt   *SystemTime `json:"expires_at,omitempty"`
	Scopes      []string    `json:"scopes,omitempty"`
}

// AuthType represents the client authentication method for the token endpoint.
type AuthType string

const (
	AuthTypeRequestBody AuthType = "requestBody"
	AuthTypeBasicAuth   AuthType = "basicAuth"
)

func (a AuthType) MarshalJSON() ([]byte, error) {
	if !a.Valid() {
		return nil, fmt.Errorf("invalid AuthType: %q", a)
	}
	return json.Marshal(string(a))
}

func (a *AuthType) UnmarshalJSON(data []byte) error {
	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		return err
	}

	at := AuthType(s)
	if !at.Valid() {
		return fmt.Errorf("invalid AuthType %q", s)
	}

	*a = at
	return nil
}

func (a AuthType) Valid() bool {
	switch a {
	case AuthTypeRequestBody, AuthTypeBasicAuth:
		return true
	default:
		return false
	}
}

// AudioContent represents audio content in a message
type AudioContent struct {
	Meta        Meta         `json:"_meta,omitempty"`
	Annotations *Annotations `json:"annotations,omitempty"`
	Data        string       `json:"data"`
	MimeType    string       `json:"mimeType"`
}

func (a AudioContent) MarshalJSON() ([]byte, error) {
	type alias AudioContent
	return json.Marshal(&struct {
		Type string `json:"type"`
		alias
	}{
		Type:  "audio",
		alias: (alias)(a),
	})
}

func (a *AudioContent) UnmarshalJSON(data []byte) error {
	type alias AudioContent
	aux := struct {
		Type string `json:"type"`
		alias
	}{}

	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}

	// Optional: validate `type`
	if aux.Type != "audio" && aux.Type != "" { // allow empty if missing
		return fmt.Errorf("invalid type %q, expected \"audio\"", aux.Type)
	}

	*a = AudioContent(aux.alias)
	return nil
}

// BlobResourceContents represents binary resource contents
type BlobResourceContents struct {
	Meta     Meta    `json:"_meta,omitempty"`
	Blob     string  `json:"blob"`
	MimeType *string `json:"mimeType,omitempty"`
	URI      string  `json:"uri"`
}

// BooleanSchema represents a boolean input schema
type BooleanSchema struct {
	Default     *bool   `json:"default,omitempty"`
	Description *string `json:"description,omitempty"`
	Title       *string `json:"title,omitempty"`
}

func (b BooleanSchema) MarshalJSON() ([]byte, error) {
	type alias BooleanSchema
	return json.Marshal(&struct {
		Type string `json:"type"`
		alias
	}{
		Type:  "boolean",
		alias: (alias)(b),
	})
}

func (b *BooleanSchema) UnmarshalJSON(data []byte) error {
	type alias BooleanSchema
	aux := struct {
		Type string `json:"type"`
		alias
	}{}

	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}

	// Optional: validate `type`
	if aux.Type != "boolean" && aux.Type != "" { // allow empty if missing
		return fmt.Errorf("invalid type %q, expected \"boolean\"", aux.Type)
	}

	*b = BooleanSchema(aux.alias)
	return nil
}

// CallToolRequest represents a request to call a tool
type CallToolRequest struct {
	Context PluginRequestContext `json:"context"`
	Request CallToolRequestParam `json:"request"`
}

// CallToolRequestParam represents parameters for calling a tool
type CallToolRequestParam struct {
	Arguments map[string]any `json:"arguments,omitempty"`
	Name      string         `json:"name"`
}

// CallToolResult represents the result of calling a tool
type CallToolResult struct {
	Meta              Meta           `json:"_meta,omitempty"`
	Content           []ContentBlock `json:"content"`
	IsError           *bool          `json:"isError,omitempty"`
	StructuredContent map[string]any `json:"structuredContent,omitempty"`
}

// Error creates an error CallToolResult with the given message.
func (CallToolResult) Error(msg string) CallToolResult {
	isErr := true
	return CallToolResult{
		IsError: &isErr,
		Content: []ContentBlock{
			{Text: &TextContent{Text: msg}},
		},
	}
}

// CompleteRequest represents a request for completion suggestions
type CompleteRequest struct {
	Context PluginRequestContext `json:"context"`
	Request CompleteRequestParam `json:"request"`
}

// CompleteRequestParam represents parameters for completion
type CompleteRequestParam struct {
	Argument CompleteRequestParamArgument `json:"argument"`
	Context  *CompleteRequestParamContext `json:"context,omitempty"`
	Ref      Reference                    `json:"ref"`
}

// CompleteRequestParamArgument represents an argument for completion
type CompleteRequestParamArgument struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

// CompleteRequestParamContext represents context for completion
type CompleteRequestParamContext struct {
	Arguments map[string]string `json:"arguments,omitempty"`
}

// CompleteResult represents completion suggestions
type CompleteResult struct {
	Completion CompleteResultCompletion `json:"completion"`
}

// CompleteResultCompletion represents completion values
type CompleteResultCompletion struct {
	HasMore *bool    `json:"hasMore,omitempty"`
	Total   *int64   `json:"total,omitempty"`
	Values  []string `json:"values"`
}

type ContentBlock struct {
	Audio            *AudioContent
	EmbeddedResource *EmbeddedResource
	Image            *ImageContent
	ResourceLink     *ResourceLinkContent
	Text             *TextContent
}

func (c ContentBlock) MarshalJSON() ([]byte, error) {
	switch {
	case c.Audio != nil:
		return json.Marshal(c.Audio)
	case c.EmbeddedResource != nil:
		return json.Marshal(c.EmbeddedResource)
	case c.Image != nil:
		return json.Marshal(c.Image)
	case c.ResourceLink != nil:
		return json.Marshal(c.ResourceLink)
	case c.Text != nil:
		return json.Marshal(c.Text)
	default:
		return nil, fmt.Errorf("empty ContentItem")
	}
}

func (c *ContentBlock) UnmarshalJSON(data []byte) error {
	var head struct {
		Type string `json:"type"`
	}
	if err := json.Unmarshal(data, &head); err != nil {
		return err
	}

	switch head.Type {
	case "audio":
		var a AudioContent
		if err := json.Unmarshal(data, &a); err != nil {
			return err
		}
		c.Audio = &a
	case "resource":
		var r EmbeddedResource
		if err := json.Unmarshal(data, &r); err != nil {
			return err
		}
		c.EmbeddedResource = &r
	case "image":
		var i ImageContent
		if err := json.Unmarshal(data, &i); err != nil {
			return err
		}
		c.Image = &i
	case "resource_link":
		var rl ResourceLinkContent
		if err := json.Unmarshal(data, &rl); err != nil {
			return err
		}
		c.ResourceLink = &rl
	case "text":
		var t TextContent
		if err := json.Unmarshal(data, &t); err != nil {
			return err
		}
		c.Text = &t
	default:
		return fmt.Errorf("unknown content type %q", head.Type)
	}

	return nil
}

// CreateMessageRequestParam represents a request to create a message
type CreateMessageRequestParam struct {
	IncludeContext   *CreateMessageRequestParamIncludeContext `json:"includeContext,omitempty"`
	MaxTokens        int64                                    `json:"maxTokens"`
	Messages         []SamplingMessage                        `json:"messages"`
	Metadata         any                                      `json:"metadata,omitempty"`
	ModelPreferences *ModelPreferences                        `json:"modelPreferences,omitempty"`
	StopSequences    []string                                 `json:"stopSequences,omitempty"`
	SystemPrompt     *string                                  `json:"systemPrompt,omitempty"`
	Task             map[string]any                           `json:"task,omitempty"`
	Temperature      *float64                                 `json:"temperature,omitempty"`
	ToolChoice       *ToolChoice                              `json:"toolChoice,omitempty"`
	Tools            []Tool                                   `json:"tools,omitempty"`
}

// CreateMessageRequestParamIncludeContext represents context inclusion options
type CreateMessageRequestParamIncludeContext string

const (
	CreateMessageRequestParamIncludeContextAllServers CreateMessageRequestParamIncludeContext = "allServers"
	CreateMessageRequestParamIncludeContextNone       CreateMessageRequestParamIncludeContext = "none"
	CreateMessageRequestParamIncludeContextThisServer CreateMessageRequestParamIncludeContext = "thisServer"
)

func (e CreateMessageRequestParamIncludeContext) MarshallJSON() ([]byte, error) {
	if !e.Valid() {
		return nil, fmt.Errorf("invalid CreateMessageRequestParamIncludeContext: %q", e)
	}
	return json.Marshal(string(e))
}

func (e *CreateMessageRequestParamIncludeContext) UnmarshalJSON(data []byte) error {
	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		return err
	}

	ct := CreateMessageRequestParamIncludeContext(s)
	if !ct.Valid() {
		return fmt.Errorf("invalid CreateMessageRequestParamIncludeContext %q", s)
	}

	*e = ct
	return nil
}

func (e CreateMessageRequestParamIncludeContext) Valid() bool {
	switch e {
	case CreateMessageRequestParamIncludeContextAllServers, CreateMessageRequestParamIncludeContextNone, CreateMessageRequestParamIncludeContextThisServer:
		return true
	default:
		return false
	}
}

// CreateMessageResult represents the result of creating a message
type CreateMessageResult struct {
	Content    CreateMessageResultContent `json:"content"`
	Model      string                     `json:"model"`
	Role       Role                       `json:"role"`
	StopReason *string                    `json:"stopReason,omitempty"`
}

type CreateMessageResultContent SamplingMessage

type Mode string

const (
	ModeForm Mode = "form"
	ModeURL  Mode = "url"
)

type FormElicitationRequestParam struct {
	Message         string `json:"message"`
	RequestedSchema Schema `json:"requestedSchema"`
}

func (FormElicitationRequestParam) Mode() Mode                 { return ModeForm }
func (FormElicitationRequestParam) isElicitationRequestParam() {}

type URLElicitationRequestParam struct {
	ElicitationID string `json:"elicitationId"`
	Message       string `json:"message"`
	URL           string `json:"url"`
}

func (URLElicitationRequestParam) Mode() Mode                 { return ModeURL }
func (URLElicitationRequestParam) isElicitationRequestParam() {}

// ElicitationRequestParam is an interface implemented by all variants.
// This mirrors your Rust enum.
type ElicitationRequestParam interface {
	Mode() Mode
	isElicitationRequestParam()
}

// ElicitationRequestParamWithTimeout represents a request for user elicitation
type ElicitationRequestParamWithTimeout struct {
	Inner   ElicitationRequestParam
	Timeout *int64
}

func (w ElicitationRequestParamWithTimeout) MarshalJSON() ([]byte, error) {
	if w.Inner == nil {
		w.Inner = FormElicitationRequestParam{
			Message:         "",
			RequestedSchema: Schema{},
		}
	}

	// Start with the mode tag.
	m := map[string]any{
		"mode": string(w.Inner.Mode()),
	}

	// Merge variant fields.
	var variantBytes []byte
	var err error
	switch v := w.Inner.(type) {
	case FormElicitationRequestParam:
		variantBytes, err = json.Marshal(v)
	case *FormElicitationRequestParam:
		variantBytes, err = json.Marshal(v)
	case URLElicitationRequestParam:
		variantBytes, err = json.Marshal(v)
	case *URLElicitationRequestParam:
		variantBytes, err = json.Marshal(v)
	default:
		return nil, fmt.Errorf("unknown Inner type %T", w.Inner)
	}
	if err != nil {
		return nil, err
	}

	var variantMap map[string]any
	if err := json.Unmarshal(variantBytes, &variantMap); err != nil {
		return nil, err
	}
	for k, val := range variantMap {
		// Prevent collisions with "mode" (shouldn't happen if your structs don't include it)
		if k == "mode" {
			continue
		}
		m[k] = val
	}

	// Add timeout (skip if nil, matching skip_serializing_if)
	if w.Timeout != nil {
		m["timeout"] = *w.Timeout
	}

	return json.Marshal(m)
}

// UnmarshalJSON reads the flat object, inspects "mode", then decodes into the right variant.
// It also extracts timeout if present.
func (w *ElicitationRequestParamWithTimeout) UnmarshalJSON(data []byte) error {
	// Decode into a generic map so we can inspect mode and pluck timeout.
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}

	// Mode is required for the union.
	modeRaw, ok := raw["mode"]
	if !ok {
		return errors.New(`missing required field "mode"`)
	}
	var mode string
	if err := json.Unmarshal(modeRaw, &mode); err != nil {
		return fmt.Errorf(`invalid "mode": %w`, err)
	}

	// Timeout is optional.
	if tRaw, ok := raw["timeout"]; ok {
		var t int64
		if err := json.Unmarshal(tRaw, &t); err != nil {
			return fmt.Errorf(`invalid "timeout": %w`, err)
		}
		w.Timeout = &t
	} else {
		w.Timeout = nil
	}

	// Remove wrapper-only fields so we can decode the remainder into the variant struct.
	delete(raw, "timeout")
	// Keep "mode" out of variant decoding; variant structs don't include it.
	delete(raw, "mode")

	// Re-marshal remaining fields into JSON for variant decoding.
	// (This is a common, reliable pattern.)
	rest, err := json.Marshal(raw)
	if err != nil {
		return err
	}

	decode_non_empty := func(b []byte, dst any) error {
		// Normalize whitespace
		if len(bytes.TrimSpace(b)) == 0 {
			return nil
		}
		return json.Unmarshal(b, dst)
	}

	switch Mode(mode) {
	case ModeForm:
		var v FormElicitationRequestParam
		if err := decode_non_empty(rest, &v); err != nil {
			return fmt.Errorf("decode form param: %w", err)
		}
		w.Inner = v
		return nil

	case ModeURL:
		var v URLElicitationRequestParam
		if err := decode_non_empty(rest, &v); err != nil {
			return fmt.Errorf("decode url param: %w", err)
		}
		w.Inner = v
		return nil

	default:
		return fmt.Errorf(`unknown mode %q`, mode)
	}
}

type ElicitationResponseNotificationParam struct {
	ElicitationID string `json:"elicitationId"`
}

// ElicitationResult represents the result of an elicitation
type ElicitationResult struct {
	Action  ElicitationResultAction                  `json:"action"`
	Content map[string]ElicitationResultContentValue `json:"content,omitempty"`
}

// ElicitationResultAction represents the action taken in elicitation
type ElicitationResultAction string

const (
	ElicitationResultActionAccept  ElicitationResultAction = "accept"
	ElicitationResultActionCancel  ElicitationResultAction = "cancel"
	ElicitationResultActionDecline ElicitationResultAction = "decline"
)

func (e ElicitationResultAction) MarshallJSON() ([]byte, error) {
	if !e.Valid() {
		return nil, fmt.Errorf("invalid ElicitationResultAction: %q", e)
	}
	return json.Marshal(string(e))
}

func (e *ElicitationResultAction) UnmarshalJSON(data []byte) error {
	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		return err
	}

	ea := ElicitationResultAction(s)
	if !ea.Valid() {
		return fmt.Errorf("invalid ElicitResultAction %q", s)
	}

	*e = ea
	return nil
}

func (e ElicitationResultAction) Valid() bool {
	switch e {
	case ElicitationResultActionAccept, ElicitationResultActionCancel, ElicitationResultActionDecline:
		return true
	default:
		return false
	}
}

type ElicitationResultContentValue struct {
	String  *string
	Number  *json.Number
	Boolean *bool
}

func (v ElicitationResultContentValue) MarshalJSON() ([]byte, error) {
	switch {
	case v.String != nil:
		return json.Marshal(v.String)
	case v.Number != nil:
		return json.Marshal(v.Number)
	case v.Boolean != nil:
		return json.Marshal(v.Boolean)
	default:
		return nil, fmt.Errorf("ElicitResultContentValue has no value set")
	}
}

func (v *ElicitationResultContentValue) UnmarshalJSON(data []byte) error {
	// Clear existing values
	*v = ElicitationResultContentValue{}

	// Try string first
	var s string
	if err := json.Unmarshal(data, &s); err == nil {
		v.String = &s
		return nil
	}

	// Then bool
	var b bool
	if err := json.Unmarshal(data, &b); err == nil {
		v.Boolean = &b
		return nil
	}

	// Then number
	var n json.Number
	if err := json.Unmarshal(data, &n); err == nil {
		v.Number = &n
		return nil
	}

	// If all fail, it's not a valid primitive for this type
	return fmt.Errorf("ElicitResultContentValue: unsupported JSON value: %s", string(data))
}

// EmbeddedResource represents an embedded resource
type EmbeddedResource struct {
	Meta        Meta             `json:"_meta,omitempty"`
	Annotations *Annotations     `json:"annotations,omitempty"`
	Resource    ResourceContents `json:"resource"`
}

func (e EmbeddedResource) MarshalJSON() ([]byte, error) {
	type alias EmbeddedResource

	return json.Marshal(&struct {
		Type string `json:"type"`
		alias
	}{
		Type:  "resource",
		alias: (alias)(e),
	})
}

func (e *EmbeddedResource) UnmarshalJSON(data []byte) error {
	type alias EmbeddedResource
	aux := struct {
		Type string `json:"type"`
		alias
	}{}

	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}

	if aux.Type != "resource" && aux.Type != "" {
		return fmt.Errorf("invalid type %q, expected \"resource\"", aux.Type)
	}

	*e = EmbeddedResource(aux.alias)
	return nil
}

// EnumSchema represents an enum input schema
type EnumSchema struct {
	Description *string  `json:"description,omitempty"`
	Enum        []string `json:"enum"`
	EnumNames   []string `json:"enumNames,omitempty"`
	Title       *string  `json:"title,omitempty"`
}

func (e EnumSchema) MarshallJSON() ([]byte, error) {
	type alias EnumSchema

	return json.Marshal(&struct {
		Type string `json:"type"`
		alias
	}{
		Type:  "resource",
		alias: (alias)(e),
	})
}

func (e *EnumSchema) UnmarshalJSON(data []byte) error {
	type alias EnumSchema
	aux := struct {
		Type string `json:"type"`
		alias
	}{}

	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}

	if aux.Type != "string" && aux.Type != "" {
		return fmt.Errorf("invalid type %q, expected \"string\"", aux.Type)
	}

	*e = EnumSchema(aux.alias)
	return nil
}

// GetPromptRequest represents a request to get a prompt
type GetPromptRequest struct {
	Context PluginRequestContext  `json:"context"`
	Request GetPromptRequestParam `json:"request"`
}

// GetPromptRequestParam represents parameters for getting a prompt
type GetPromptRequestParam struct {
	Arguments map[string]string `json:"arguments,omitempty"`
	Name      string            `json:"name"`
}

// GetPromptResult represents the result of getting a prompt
type GetPromptResult struct {
	Description *string         `json:"description,omitempty"`
	Messages    []PromptMessage `json:"messages"`
}

// ImageContent represents image content
type ImageContent struct {
	Meta        Meta         `json:"_meta,omitempty"`
	Annotations *Annotations `json:"annotations,omitempty"`
	Data        string       `json:"data"`
	MimeType    string       `json:"mimeType"`
}

func (i ImageContent) MarshallJSON() ([]byte, error) {
	type alias ImageContent

	return json.Marshal(&struct {
		Type string `json:"type"`
		alias
	}{
		Type:  "image",
		alias: (alias)(i),
	})
}

func (i *ImageContent) UnmarshalJSON(data []byte) error {
	type alias ImageContent
	aux := struct {
		Type string `json:"type"`
		alias
	}{}

	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}

	if aux.Type != "image" && aux.Type != "" { // allow empty if missing
		return fmt.Errorf("invalid type %q, expected \"image\"", aux.Type)
	}

	*i = ImageContent(aux.alias)
	return nil
}

type KeyringEntryId struct {
	Service string `json:"service"`
	User    string `json:"user"`
}

// ListPromptsRequest represents a request to list prompts
type ListPromptsRequest struct {
	Context PluginRequestContext `json:"context"`
}

// ListPromptsResult represents the result of listing prompts
type ListPromptsResult struct {
	Prompts []Prompt `json:"prompts"`
}

// ListResourcesRequest represents a request to list resources
type ListResourcesRequest struct {
	Context PluginRequestContext `json:"context"`
}

// ListResourcesResult represents the result of listing resources
type ListResourcesResult struct {
	Resources []Resource `json:"resources"`
}

// ListResourceTemplatesRequest represents a request to list resource templates
type ListResourceTemplatesRequest struct {
	Context PluginRequestContext `json:"context"`
}

// ListResourceTemplatesResult represents the result of listing resource templates
type ListResourceTemplatesResult struct {
	ResourceTemplates []ResourceTemplate `json:"resourceTemplates"`
}

// ListRootsResult represents the result of listing roots
type ListRootsResult struct {
	Roots []Root `json:"roots"`
}

// ListToolsRequest represents a request to list tools
type ListToolsRequest struct {
	Context PluginRequestContext `json:"context"`
}

// ListToolsResult represents the result of listing tools
type ListToolsResult struct {
	Tools []Tool `json:"tools"`
}

// LoggingLevel represents the severity level of a log message
type LoggingLevel string

const (
	LoggingLevelDebug     LoggingLevel = "debug"
	LoggingLevelInfo      LoggingLevel = "info"
	LoggingLevelNotice    LoggingLevel = "notice"
	LoggingLevelWarning   LoggingLevel = "warning"
	LoggingLevelError     LoggingLevel = "error"
	LoggingLevelCritical  LoggingLevel = "critical"
	LoggingLevelAlert     LoggingLevel = "alert"
	LoggingLevelEmergency LoggingLevel = "emergency"
)

func (l LoggingLevel) MarshallJSON() ([]byte, error) {
	if !l.Valid() {
		return nil, fmt.Errorf("invalid LoggingLevel: %q", l)
	}
	return json.Marshal(string(l))
}

func (l *LoggingLevel) UnmarshalJSON(data []byte) error {
	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		return err
	}

	ll := LoggingLevel(s)
	if !ll.Valid() {
		return fmt.Errorf("invalid LoggingLevel %q", s)
	}

	*l = ll
	return nil
}

func (l LoggingLevel) Valid() bool {
	switch l {
	case LoggingLevelDebug, LoggingLevelInfo, LoggingLevelNotice, LoggingLevelWarning, LoggingLevelError, LoggingLevelCritical, LoggingLevelAlert, LoggingLevelEmergency:
		return true
	default:
		return false
	}
}

// LoggingMessageNotificationParam represents a logging message notification
type LoggingMessageNotificationParam struct {
	Data   any          `json:"data"`
	Level  LoggingLevel `json:"level"`
	Logger *string      `json:"logger,omitempty"`
}

// Meta represents metadata as a generic JSON object
type Meta map[string]any

// ModelHint represents a hint for model selection
type ModelHint struct {
	Name string `json:"name"`
}

// ModelPreferences represents preferences for model selection
type ModelPreferences struct {
	CostPriority         float32     `json:"costPriority,omitempty"`
	Hints                []ModelHint `json:"hints,omitempty"`
	IntelligencePriority float32     `json:"intelligencePriority,omitempty"`
	SpeedPriority        float32     `json:"speedPriority,omitempty"`
}

// NumberSchema represents a number input schema
type NumberSchema struct {
	Description *string    `json:"description,omitempty"`
	Maximum     *float64   `json:"maximum,omitempty"`
	Minimum     *float64   `json:"minimum,omitempty"`
	Title       *string    `json:"title,omitempty"`
	Type        NumberType `json:"type"` // "number" or "integer"
}

// NumberType represents the type of a number schema
type NumberType string

const (
	NumberTypeNumber  NumberType = "number"
	NumberTypeInteger NumberType = "integer"
)

func (n NumberType) MarshallJSON() ([]byte, error) {
	if !n.Valid() {
		return nil, fmt.Errorf("invalid NumberType: %q", n)
	}
	return json.Marshal(string(n))
}

func (n *NumberType) UnmarshalJSON(data []byte) error {
	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		return err
	}

	nt := NumberType(s)
	if !nt.Valid() {
		return fmt.Errorf("invalid NumberType %q", s)
	}

	*n = nt
	return nil
}

func (n NumberType) Valid() bool {
	switch n {
	case NumberTypeNumber, NumberTypeInteger:
		return true
	default:
		return false
	}
}

// OauthCredentials contains the credentials needed to obtain an OAuth2 access token from the host.
// Pass this to GetAccessToken and the host will return a cached or freshly-acquired AccessToken.
type OauthCredentials struct {
	AuthType               *AuthType         `json:"auth_type,omitempty"`
	ClientID               string            `json:"client_id"`
	ClientSecret           *string           `json:"client_secret,omitempty"`
	DeviceAuthorizationURL *string           `json:"device_authorization_url,omitempty"`
	DeviceAuthTimeoutSecs  *uint64           `json:"device_auth_timeout_secs,omitempty"`
	ExtraParams            map[string]string `json:"extra_params,omitempty"`
	Scopes                 []string          `json:"scopes,omitempty"`
	TokenEndpointURL       string            `json:"token_endpoint_url"`
}

// PluginNotificationContext represents the context for a plugin notification
type PluginNotificationContext struct {
	Meta Meta `json:"meta"`
}

// PluginRequestContext represents the context for a plugin request
type PluginRequestContext struct {
	Meta Meta            `json:"_meta"`
	ID   PluginRequestId `json:"id"`
}

type PluginRequestId struct {
	String *string
	Number *int64
}

func (p PluginRequestId) MarshalJSON() ([]byte, error) {
	switch {
	case p.String != nil:
		return json.Marshal(p.String)
	case p.Number != nil:
		return json.Marshal(p.Number)
	default:
		return nil, fmt.Errorf("empty PluginRequestId")
	}
}

func (p *PluginRequestId) UnmarshalJSON(data []byte) error {
	*p = PluginRequestId{}

	// Try string first
	var s string
	if err := json.Unmarshal(data, &s); err == nil {
		p.String = &s
		return nil
	}

	// Then number
	var n int64
	if err := json.Unmarshal(data, &n); err == nil {
		p.Number = &n
		return nil
	}

	// If all fail, it's not a valid primitive for this type
	return fmt.Errorf("PluginRequestId: unsupported JSON value: %s", string(data))
}

// PrimitiveSchemaDefinition is a union type for schema definitions
type PrimitiveSchemaDefinition struct {
	Boolean *BooleanSchema
	Enum    *EnumSchema
	Number  *NumberSchema
	String  *StringSchema
}

func (p PrimitiveSchemaDefinition) MarshalJSON() ([]byte, error) {
	switch {
	case p.Boolean != nil:
		return json.Marshal(p.Boolean)
	case p.Enum != nil:
		return json.Marshal(p.Enum)
	case p.Number != nil:
		return json.Marshal(p.Number)
	case p.String != nil:
		return json.Marshal(p.String)
	default:
		return nil, fmt.Errorf("empty PrimitiveSchemaDefinition")
	}
}

func (p *PrimitiveSchemaDefinition) UnmarshalJSON(data []byte) error {
	var head struct {
		Type string `json:"type"`
	}
	if err := json.Unmarshal(data, &head); err != nil {
		return err
	}

	switch head.Type {
	case "boolean":
		var b BooleanSchema
		if err := json.Unmarshal(data, &b); err != nil {
			return err
		}
		p.Boolean = &b
	case "string":
		var e EnumSchema
		if err := json.Unmarshal(data, &e); err != nil {
			var s StringSchema
			if err := json.Unmarshal(data, &s); err != nil {
				return err
			}
			p.String = &s
		} else {
			p.Enum = &e
		}
	case "number", "integer":
		var n NumberSchema
		if err := json.Unmarshal(data, &n); err != nil {
			return err
		}
		p.Number = &n
	}

	return nil
}

// ProgressNotificationParam represents a progress notification
type ProgressNotificationParam struct {
	Message       *string  `json:"message,omitempty"`
	Progress      float64  `json:"progress"`
	ProgressToken string   `json:"progressToken"`
	Total         *float64 `json:"total,omitempty"`
}

// Prompt represents a prompt
type Prompt struct {
	Arguments   []PromptArgument `json:"arguments,omitempty"`
	Description *string          `json:"description,omitempty"`
	Name        string           `json:"name"`
	Title       *string          `json:"title,omitempty"`
}

// PromptArgument represents an argument for a prompt
type PromptArgument struct {
	Description *string `json:"description,omitempty"`
	Name        string  `json:"name"`
	Required    *bool   `json:"required,omitempty"`
	Title       *string `json:"title,omitempty"`
}

// PromptMessage represents a message in a prompt
type PromptMessage struct {
	Content ContentBlock `json:"content"`
	Role    Role         `json:"role"`
}

// PromptReference represents a reference to a prompt
type PromptReference struct {
	Name  string  `json:"name"`
	Title *string `json:"title,omitempty"`
}

func (p PromptReference) MarshalJSON() ([]byte, error) {
	type alias PromptReference
	return json.Marshal(&struct {
		Type string `json:"type"`
		alias
	}{
		Type:  "prompt",
		alias: (alias)(p),
	})
}

func (p *PromptReference) UnmarshalJSON(data []byte) error {
	type alias PromptReference
	aux := struct {
		Type string `json:"type"`
		alias
	}{}

	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}

	if aux.Type != "prompt" && aux.Type != "" { // allow empty if missing
		return fmt.Errorf("invalid type %q, expected \"prompt\"", aux.Type)
	}

	*p = PromptReference(aux.alias)
	return nil
}

// ReadResourceRequest represents a request to read a resource
type ReadResourceRequest struct {
	Context PluginRequestContext     `json:"context"`
	Request ReadResourceRequestParam `json:"request"`
}

// ReadResourceRequestParam represents parameters for reading a resource
type ReadResourceRequestParam struct {
	URI string `json:"uri"`
}

// ReadResourceResult represents the result of reading a resource
type ReadResourceResult struct {
	Contents []ResourceContents `json:"contents"`
}

type Reference struct {
	Prompt           *PromptReference
	ResourceTemplate *ResourceTemplateReference
}

func (r Reference) MarshalJSON() ([]byte, error) {
	switch {
	case r.Prompt != nil:
		return json.Marshal(r.Prompt)
	case r.ResourceTemplate != nil:
		return json.Marshal(r.ResourceTemplate)
	default:
		return nil, fmt.Errorf("empty Reference")
	}
}

func (r *Reference) UnmarshalJSON(data []byte) error {
	var head struct {
		Type string `json:"type"`
	}
	if err := json.Unmarshal(data, &head); err != nil {
		return err
	}

	switch head.Type {
	case "prompt":
		var p PromptReference
		if err := json.Unmarshal(data, &p); err != nil {
			return err
		}
		r.Prompt = &p
	case "resource":
		var rt ResourceTemplateReference
		if err := json.Unmarshal(data, &rt); err != nil {
			return err
		}
		r.ResourceTemplate = &rt
	default:
		return fmt.Errorf("unknown reference type %q", head.Type)
	}

	return nil
}

// Resource represents a resource
type Resource struct {
	Annotations *Annotations `json:"annotations,omitempty"`
	Description *string      `json:"description,omitempty"`
	MimeType    *string      `json:"mimeType,omitempty"`
	Name        string       `json:"name"`
	Size        *int64       `json:"size,omitempty"`
	Title       *string      `json:"title,omitempty"`
	URI         string       `json:"uri"`
}

type ResourceContents struct {
	Blob *BlobResourceContents
	Text *TextResourceContents
}

func (R ResourceContents) MarshalJSON() ([]byte, error) {
	switch {
	case R.Blob != nil:
		return json.Marshal(R.Blob)
	case R.Text != nil:
		return json.Marshal(R.Text)
	default:
		return nil, fmt.Errorf("empty ResourceContents")
	}
}

func (r *ResourceContents) UnmarshalJSON(data []byte) error {
	// Clear existing values
	*r = ResourceContents{}

	// Try blob first
	var b BlobResourceContents
	if err := json.Unmarshal(data, &b); err == nil {
		r.Blob = &b
		return nil
	}

	// Then text
	var t TextResourceContents
	if err := json.Unmarshal(data, &t); err == nil {
		r.Text = &t
		return nil
	}

	// If all fail, it's not a valid ResourceContents
	return fmt.Errorf("ResourceContents: unsupported JSON value: %s", string(data))
}

// ResourceLinkContent represents a link to a resource
type ResourceLinkContent struct {
	Meta        Meta         `json:"_meta,omitempty"`
	Annotations *Annotations `json:"annotations,omitempty"`
	Description *string      `json:"description,omitempty"`
	MimeType    *string      `json:"mimeType,omitempty"`
	Name        string       `json:"name"`
	Size        *int64       `json:"size,omitempty"`
	Title       *string      `json:"title,omitempty"`
	URI         string       `json:"uri"`
}

func (r ResourceLinkContent) MarshallJSON() ([]byte, error) {
	type alias ResourceLinkContent
	return json.Marshal(&struct {
		Type string `json:"type"`
		alias
	}{
		Type:  "resource_link",
		alias: (alias)(r),
	})
}

func (r *ResourceLinkContent) UnmarshalJSON(data []byte) error {
	type alias ResourceLinkContent
	aux := struct {
		Type string `json:"type"`
		alias
	}{}

	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}

	if aux.Type != "resource_link" && aux.Type != "" { // allow empty if missing
		return fmt.Errorf("invalid type %q, expected \"resource_link\"", aux.Type)
	}

	*r = ResourceLinkContent(aux.alias)
	return nil
}

// ResourceTemplate represents a resource template
type ResourceTemplate struct {
	Annotations *Annotations `json:"annotations,omitempty"`
	Description *string      `json:"description,omitempty"`
	MimeType    *string      `json:"mimeType,omitempty"`
	Name        string       `json:"name"`
	Title       *string      `json:"title,omitempty"`
	URITemplate string       `json:"uriTemplate"`
}

// ResourceTemplateReference represents a reference to a resource template
type ResourceTemplateReference struct {
	URI string `json:"uri"`
}

func (r ResourceTemplateReference) MarshallJSON() ([]byte, error) {
	type alias ResourceTemplateReference
	return json.Marshal(&struct {
		Type string `json:"type"`
		alias
	}{
		Type:  "resource",
		alias: (alias)(r),
	})
}

func (r *ResourceTemplateReference) UnmarshalJSON(data []byte) error {
	type alias ResourceTemplateReference
	aux := struct {
		Type string `json:"type"`
		alias
	}{}

	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}

	if aux.Type != "resource" && aux.Type != "" { // allow empty if missing
		return fmt.Errorf("invalid type %q, expected \"resource\"", aux.Type)
	}

	*r = ResourceTemplateReference(aux.alias)
	return nil
}

// ResourceUpdatedNotificationParam represents a resource update notification
type ResourceUpdatedNotificationParam struct {
	URI string `json:"uri"`
}

// Role represents the role of a message sender
type Role string

const (
	RoleAssistant Role = "assistant"
	RoleUser      Role = "user"
)

func (r Role) MarshallJSON() ([]byte, error) {
	if !r.Valid() {
		return nil, fmt.Errorf("invalid Role: %q", r)
	}
	return json.Marshal(string(r))
}

func (r *Role) UnmarshalJSON(data []byte) error {
	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		return err
	}

	rr := Role(s)
	if !rr.Valid() {
		return fmt.Errorf("invalid Role %q", s)
	}

	*r = rr
	return nil
}

func (r Role) Valid() bool {
	switch r {
	case RoleAssistant, RoleUser:
		return true
	default:
		return false
	}
}

// Root represents a root directory or resource
type Root struct {
	Name *string `json:"name,omitempty"`
	URI  string  `json:"uri"`
}

type SamplingMessage struct {
	Audio *AudioContent
	Image *ImageContent
	Text  *TextContent
}

func (s SamplingMessage) MarshalJSON() ([]byte, error) {
	switch {
	case s.Audio != nil:
		return json.Marshal(s.Audio)
	case s.Image != nil:
		return json.Marshal(s.Image)
	case s.Text != nil:
		return json.Marshal(s.Text)
	default:
		return nil, fmt.Errorf("empty SamplingMessage")
	}
}

func (s *SamplingMessage) UnmarshalJSON(data []byte) error {
	var head struct {
		Type string `json:"type"`
	}
	if err := json.Unmarshal(data, &head); err != nil {
		return err
	}

	switch head.Type {
	case "audio":
		var a AudioContent
		if err := json.Unmarshal(data, &a); err != nil {
			return err
		}
		s.Audio = &a
	case "image":
		var i ImageContent
		if err := json.Unmarshal(data, &i); err != nil {
			return err
		}
		s.Image = &i
	case "text":
		var t TextContent
		if err := json.Unmarshal(data, &t); err != nil {
			return err
		}
		s.Text = &t
	default:
		return fmt.Errorf("unknown content type %q", head.Type)
	}

	return nil
}

// Schema represents a JSON schema
type Schema struct {
	Properties map[string]PrimitiveSchemaDefinition `json:"properties,omitempty"`
	Required   []string                             `json:"required,omitempty"`
}

func (s Schema) MarshallJSON() ([]byte, error) {
	type alias Schema
	return json.Marshal(&struct {
		Type string `json:"type"`
		alias
	}{
		Type:  "object",
		alias: (alias)(s),
	})
}

func (s *Schema) UnmarshalJSON(data []byte) error {
	type alias Schema
	aux := struct {
		Type string `json:"type"`
		alias
	}{}

	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}

	// Optional: validate `type`
	if aux.Type != "object" && aux.Type != "" { // allow empty if missing
		return fmt.Errorf("invalid type %q, expected \"object\"", aux.Type)
	}

	*s = Schema(aux.alias)
	return nil
}

// StringSchema represents a string input schema
type StringSchema struct {
	Description *string             `json:"description,omitempty"`
	Format      *StringSchemaFormat `json:"format,omitempty"`
	MaxLength   *int64              `json:"maxLength,omitempty"`
	MinLength   *int64              `json:"minLength,omitempty"`
	Title       *string             `json:"title,omitempty"`
}

func (s StringSchema) MarshallJSON() ([]byte, error) {
	type alias StringSchema
	return json.Marshal(&struct {
		Type string `json:"type"`
		alias
	}{
		Type:  "string",
		alias: (alias)(s),
	})
}

func (s *StringSchema) UnmarshalJSON(data []byte) error {
	type alias StringSchema
	aux := struct {
		Type string `json:"type"`
		alias
	}{}

	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}

	// Optional: validate `type`
	if aux.Type != "string" && aux.Type != "" { // allow empty if missing
		return fmt.Errorf("invalid type %q, expected \"string\"", aux.Type)
	}

	*s = StringSchema(aux.alias)
	return nil
}

// StringSchemaFormat represents the format of a string schema
type StringSchemaFormat string

const (
	Email    StringSchemaFormat = "email"
	URI      StringSchemaFormat = "uri"
	Date     StringSchemaFormat = "date"
	DateTime StringSchemaFormat = "date_time"
)

func (s StringSchemaFormat) MarshallJSON() ([]byte, error) {
	if !s.Valid() {
		return nil, fmt.Errorf("invalid StringSchemaFormat: %q", s)
	}
	return json.Marshal(string(s))
}

func (s *StringSchemaFormat) UnmarshalJSON(data []byte) error {
	var str string
	if err := json.Unmarshal(data, &str); err != nil {
		return err
	}

	sf := StringSchemaFormat(str)
	if !sf.Valid() {
		return fmt.Errorf("invalid StringSchemaFormat %q", str)
	}

	*s = sf
	return nil
}

func (s StringSchemaFormat) Valid() bool {
	switch s {
	case Email, URI, Date, DateTime:
		return true
	default:
		return false
	}
}

// SystemTime mirrors Rust's std::time::SystemTime serde representation.
type SystemTime struct {
	SecsSinceEpoch  uint64 `json:"secs_since_epoch"`
	NanosSinceEpoch uint32 `json:"nanos_since_epoch"`
}

// TextContent represents text content
type TextContent struct {
	Meta        Meta         `json:"_meta,omitempty"`
	Annotations *Annotations `json:"annotations,omitempty"`
	Text        string       `json:"text"`
}

func (t TextContent) MarshallJSON() ([]byte, error) {
	type alias TextContent
	return json.Marshal(&struct {
		Type string `json:"type"`
		alias
	}{
		Type:  "text",
		alias: (alias)(t),
	})
}

func (t *TextContent) UnmarshalJSON(data []byte) error {
	type alias TextContent
	aux := struct {
		Type string `json:"type"`
		alias
	}{}

	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}

	if aux.Type != "text" && aux.Type != "" { // allow empty if missing
		return fmt.Errorf("invalid type %q, expected \"text\"", aux.Type)
	}

	*t = TextContent(aux.alias)
	return nil
}

// TextResourceContents represents text resource contents
type TextResourceContents struct {
	Meta     Meta    `json:"_meta,omitempty"`
	MimeType *string `json:"mimeType,omitempty"`
	Text     string  `json:"text"`
	URI      string  `json:"uri"`
}

// Tool represents a tool
type Tool struct {
	Annotations  *ToolAnnotations   `json:"annotations,omitempty"`
	Description  *string            `json:"description,omitempty"`
	InputSchema  jsonschema.Schema  `json:"inputSchema"`
	Name         string             `json:"name"`
	OutputSchema *jsonschema.Schema `json:"outputSchema,omitempty"`
	Title        *string            `json:"title,omitempty"`
}

type ToolAnnotations struct {
	DestructiveHint *bool   `json:"destructiveHint,omitempty"`
	IdempotentHint  *bool   `json:"idempotentHint,omitempty"`
	OpenWorldHint   *bool   `json:"openWorldHint,omitempty"`
	ReadOnlyHint    *bool   `json:"readOnlyHint,omitempty"`
	Title           *string `json:"title,omitempty"`
}

type ToolChoice struct {
	Mode ToolChoiceMode `json:"mode,omitempty"`
}

type ToolChoiceMode string

const (
	ToolChoiceModeAuto     ToolChoiceMode = "auto"
	ToolChoiceModeRequired ToolChoiceMode = "required"
	ToolChoiceModeNone     ToolChoiceMode = "none"
)

func (t ToolChoiceMode) MarshallJSON() ([]byte, error) {
	if !t.Valid() {
		return nil, fmt.Errorf("invalid ToolChoiceMode: %q", t)
	}
	return json.Marshal(string(t))
}

func (t *ToolChoiceMode) UnmarshallJSON(data []byte) error {
	var str string
	if err := json.Unmarshal(data, &str); err != nil {
		return err
	}

	tcm := ToolChoiceMode(str)
	if !tcm.Valid() {
		return fmt.Errorf("invalid ToolChoiceMode %q", str)
	}

	*t = tcm
	return nil
}

func (t ToolChoiceMode) Valid() bool {
	switch t {
	case ToolChoiceModeAuto, ToolChoiceModeRequired, ToolChoiceModeNone:
		return true
	default:
		return false
	}
}
