# API Error Handling Overhaul Plan

## Goals
1. Maintain exact backward compatibility with frontend ApiError contract
2. Create plugin architecture for third-party telemetry/error platforms
3. Separate concerns: error creation, enrichment, classification, telemetry, response
4. Provide clean extension points for observability tools
5. Preserve logging-once principle with clearer semantics
6. Add contextual enrichment capabilities
7. Support metrics and monitoring hooks

## Current Issues to Address
1. HTTPError function does too much (logging + mapping + response)
2. Telemetry is hardcoded to zap - not pluggable
3. Error handling logic scattered across files
4. No centralized error policy/classification
5. Limited contextual enrichment capabilities
6. No standard metrics hooks

## Proposed Architecture

### 1. Core Error Types (Enhanced xerrors)
Keep and enhance the existing `xerrors.DomainError` but add extension points:

```go
// Enhanced DomainError with telemetry hooks
type DomainError struct {
    Code    string              `json:"code"`
    Message string              `json:"message"`
    Status  int                 `json:"-"`
    Fields  map[string][]string `json:"errors,omitempty"`
    // Optional metadata for telemetry (not sent to client)
    Meta    map[string]any      `json:"-"`
    // Source information for debugging
    Source  *ErrorSource        `json:"-"`
}

// ErrorSource captures where error originated
type ErrorSource struct {
    Package string
    Function string
    Line    int
}

// TelemetrySink interface for plugins
type TelemetrySink interface {
    // ProcessError sends error to telemetry platform
    ProcessError(ctx context.Context, err *DomainError, req *TelemetryRequest)
    // Flush ensures all errors are sent
    Flush(ctx context.Context) error
    // Name identifies the sink
    Name() string
}

// TelemetryRequest contains contextual information
type TelemetryRequest struct {
    Method   string
    Path     string
    Query    url.Values
    Headers  http.Header
    UserID   string
    TenantID string
    RequestID string
    // Any additional context
    Context  map[string]any
}
```

### 2. Error Policy Engine
Centralized error classification and handling policies:

```go
// ErrorPolicy defines how errors should be handled
type ErrorPolicy struct {
    // Should this error be logged?
    ShouldLog bool
    // Should this error be sent to telemetry?
    ShouldTelemetry bool
    // Should message be sanitized for 5xx errors?
    SanitizeMessage bool
    // Custom message for 5xx errors (if sanitizing)
    GenericMessage string
    // Enrichment functions
    Enrichers []ErrorEnricher
}

// ErrorEnricher adds contextual information
type ErrorEnricher func(ctx context.Context, err *DomainError, req *TelemetryRequest) *DomainError
```

### 3. Plugin-Based Telemetry System
Register multiple sinks that can receive errors:

```go
// TelemetryManager coordinates all sinks
type TelemetryManager struct {
    sinks []TelemetrySink
    policy *ErrorPolicy
    logger *zap.SugaredLogger
}

// Example sinks:
// - ZapSink (existing logging)
// - SentrySink
// - DatadogSink
// - PrometheusSink (for metrics)
// - CustomSink (for user-defined telemetry)
```

### 4. Enhanced Error Handling Flow

#### Error Creation (unchanged, but with options)
```go
// Services/repositories create errors as before
return xerrors.New("invalid_input", http.StatusBadRequest, "email is invalid")
// With optional metadata for telemetry
err := xerrors.New("db_timeout", http.StatusGatewayTimeout, "query timeout")
err.WithMeta(map[string]any{"query": query, "duration": duration})
```

#### Error Enrichment (new)
As errors bubble up, they can be enriched with context:
```go
// In middleware or handlers
err = enrichErrorWithRequestContext(c, err)
// Or with user context
err = enrichErrorWithUserContext(userID, tenantID, err)
```

#### Error Handling (refactored HTTPError)
```go
func HTTPError(c *fiber.Ctx, err error) error {
    // 1. Extract DomainError
    de := extractDomainError(err)
    
    // 2. Apply enrichment from policy
    de = policy.ApplyEnrichers(c, de)
    
    // 3. Create telemetry request
    teleReq := buildTelemetryRequest(c)
    
    // 4. Process through telemetry sinks (async, non-blocking)
    go telemetryManager.Process(de, teleReq)
    
    // 5. Log if policy says so (separate from telemetry)
    if policy.ShouldLog(de) {
        logger.LogError(c, de)
    }
    
    // 6. Build response (unchanged contract)
    return buildResponse(c, de)
}
```

### 5. Backward Compatibility Guarantees
- Exact same JSON response structure: `{code, message, errors, request_id}`
- Same HTTP status code mappings
- Same error codes (snake_case strings)
- Same frontend ApiError class compatibility
- Same error wrapping patterns (`%w`)

### 6. Implementation Plan

#### Phase 1: Foundation
- Enhance `xerrors.DomainError` with Meta and Source fields
- Create `TelemetrySink` interface and basic implementations
- Build `TelemetryManager` to coordinate sinks
- Create `ErrorPolicy` and `ErrorEnricher` concepts

#### Phase 2: Refactor HTTPError
- Split HTTPError into focused functions:
  - extractDomainError
  - enrichError
  - processTelemetry
  - logError (if needed)
  - buildResponse
- Make each function testable in isolation

#### Phase 3: Plugin Examples
- Implement ZapSink (maintains existing logging)
- Implement SentrySink example
- Implement PrometheusSink for error metrics
- Show how to add custom sinks

#### Phase 4: Enrichment Helpers
- Create enrichment functions for:
  - Request context (method, path, query, headers)
  - User context (user_id, tenant_id, role)
  - Session context
  - Custom business context

#### Phase 5: Testing & Documentation
- Comprehensive unit tests
- Integration tests showing plugin usage
- Updated AGENTS.md with new patterns
- Examples for common telemetry platforms

## Benefits
1. **Pluggable Telemetry**: Add Sentry, Datadog, etc. with minimal code changes
2. **Clear Separation**: Each concern (logging, telemetry, response) is separate
3. **Extensible**: Easy to add new error enrichment or processing
4. **Maintainable**: Smaller, focused functions
5. **Observable**: Better context for debugging and monitoring
6. **Compatible**: Zero breaking changes for frontend or existing code

## Usage Examples

### Adding Sentry Telemetry
```go
// In main.go or bootstrap
sentrySink := NewSentrySink(os.Getenv("SENTRY_DSN"))
telemetryManager.RegisterSink(sentrySink)
```

### Adding Custom Context
```go
// In auth handler after user lookup
err = enrichErrorWithUserContext(c, err, user.ID, tenant.ID)
// Or in service layer
err = enrichErrorWithBusinessContext(err, "payment_failed", map[string]any{
    "amount": amount,
    "currency": currency,
})
```

### Error Policy Configuration
```go
// Different policies for different error types
validationPolicy := &ErrorPolicy{
    ShouldLog: true,
    ShouldTelemetry: false, // Don't spam telemetry with validation errors
    SanitizeMessage: false,
}

internalPolicy := &ErrorPolicy{
    ShouldLog: true,
    ShouldTelemetry: true,
    SanitizeMessage: true,
    GenericMessage: "an unexpected error occurred",
}
```

## Files to Create/Modify
1. `internal/xerrors/` - Enhanced DomainError
2. `internal/telemetry/` - Sink interfaces, manager, policies
3. `internal/middleware/errors.go` - Refactored HTTPError
4. `internal/middleware/enrichment.go` - Context enrichment helpers
5. `internal/middleware/policy.go` - Error policy definitions
6. Examples in docs for common telemetry integrations

This overhaul provides a enterprise-grade error handling system that's ready for production observability while keeping the exact same contract that the frontend depends on.