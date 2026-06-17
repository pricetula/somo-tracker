package students

import (
	"bufio"
	"encoding/csv"
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/gofiber/fiber/v2"
	"go.uber.org/fx"

	"somotracker/backend/internal/auth"
)

// Handler exposes student HTTP endpoints.
type Handler struct {
	svc     *Service
	authSvc *auth.Service
}

// NewHandler creates a new Handler.
func NewHandler(svc *Service, authSvc *auth.Service) *Handler {
	return &Handler{svc: svc, authSvc: authSvc}
}

// RegisterRoutes mounts student routes on the given router.
func (h *Handler) RegisterRoutes(router fiber.Router) {
	students := router.Group("/api/v1/students")

	// Protected routes
	students.Get("/", h.requireAuth, h.List)
	students.Post("/", h.requireAuth, h.Create)
	students.Post("/import", h.requireAuth, h.Import)
	students.Get("/import/stream", h.ImportStream)
	students.Get("/import/errors", h.ImportErrors)
}

// ─── Auth middleware ───────────────────────────────────────────────────────

func (h *Handler) requireAuth(c *fiber.Ctx) error {
	token := c.Cookies("somo_sid")
	if token == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(ErrorBody{
			Error:   "unauthorized",
			Message: "no session cookie found",
		})
	}

	session, err := h.authSvc.GetSession(c.Context(), token)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(ErrorBody{
			Error:   "unauthorized",
			Message: "invalid or expired session",
		})
	}

	c.Locals("tenant_id", session.TenantID)
	c.Locals("user_id", session.UserID)
	return c.Next()
}

// ─── Handlers ──────────────────────────────────────────────────────────────

// List handles GET /api/v1/students
//
// @Summary      List students
// @Description  Returns paginated students for the authenticated tenant.
// @Tags         Students
// @Produce      json
// @Param        page     query  int     false  "Page number (1-indexed)"
// @Param        per_page query  int     false  "Items per page (max 100)"
// @Param        search   query  string  false  "Search by name"
// @Success      200  {object}  ListResponse
// @Failure      401  {object}  ErrorBody  "Unauthorized"
// @Failure      500  {object}  ErrorBody  "Internal error"
// @Router       /api/v1/students [get]
func (h *Handler) List(c *fiber.Ctx) error {
	tenantID := c.Locals("tenant_id").(string)

	page, _ := strconv.Atoi(c.Query("page", "1"))
	perPage, _ := strconv.Atoi(c.Query("per_page", "50"))
	search := strings.TrimSpace(c.Query("search", ""))

	if page < 1 {
		page = 1
	}
	if perPage < 1 || perPage > 100 {
		perPage = 50
	}

	offset := (page - 1) * perPage

	students, total, err := h.svc.ListStudents(c.Context(), tenantID, offset, perPage, search)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(ErrorBody{
			Error:   "internal_error",
			Message: err.Error(),
		})
	}

	return c.JSON(ListResponse{
		Students: students,
		Total:    total,
	})
}

// Create handles POST /api/v1/students
//
// @Summary      Create a student
// @Description  Creates a single student record manually.
// @Tags         Students
// @Accept       json
// @Produce      json
// @Param        body  body      CreateStudentPayload  true  "Student details"
// @Success      201   {object}  Student
// @Failure      400   {object}  ErrorBody  "Invalid input"
// @Failure      401   {object}  ErrorBody  "Unauthorized"
// @Failure      500   {object}  ErrorBody  "Internal error"
// @Router       /api/v1/students [post]
func (h *Handler) Create(c *fiber.Ctx) error {
	tenantID := c.Locals("tenant_id").(string)

	var payload CreateStudentPayload
	if err := c.BodyParser(&payload); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(ErrorBody{
			Error:   "invalid_input",
			Message: "invalid request body",
		})
	}

	student, err := h.svc.CreateStudent(c.Context(), tenantID, payload)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(ErrorBody{
			Error:   "invalid_input",
			Message: err.Error(),
		})
	}

	return c.Status(fiber.StatusCreated).JSON(student)
}

// Import handles POST /api/v1/students/import
//
// @Summary      Import students via CSV
// @Description  Accepts a CSV file upload, starts background ingestion, returns an import_id.
// @Tags         Students
// @Accept       multipart/form-data
// @Produce      json
// @Param        file  formData  file  true  "CSV file with student data"
// @Success      202   {object}  ImportResponse
// @Failure      400   {object}  ErrorBody  "Invalid input"
// @Failure      401   {object}  ErrorBody  "Unauthorized"
// @Failure      500   {object}  ErrorBody  "Internal error"
// @Router       /api/v1/students/import [post]
func (h *Handler) Import(c *fiber.Ctx) error {
	tenantID := c.Locals("tenant_id").(string)

	// Parse multipart form — max 32MB
	file, err := c.FormFile("file")
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(ErrorBody{
			Error:   "invalid_input",
			Message: "file is required",
		})
	}

	// Validate file type by extension and content-type
	if !strings.HasSuffix(strings.ToLower(file.Filename), ".csv") {
		return c.Status(fiber.StatusBadRequest).JSON(ErrorBody{
			Error:   "invalid_input",
			Message: "only CSV files are accepted",
		})
	}

	f, err := file.Open()
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(ErrorBody{
			Error:   "internal_error",
			Message: "failed to open uploaded file",
		})
	}
	defer func() { _ = f.Close() }()

	// Quick pre-flight header validation before spinning up goroutine
	reader := csv.NewReader(f)
	reader.TrimLeadingSpace = true
	header, err := reader.Read()
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(ErrorBody{
			Error:   "invalid_input",
			Message: "failed to read CSV header row",
		})
	}

	normalizedHeader := normalizeHeaders(header)
	if !validateHeaders(normalizedHeader) {
		var missing []string
		for _, expected := range expectedCSVHeaders {
			found := false
			for _, h := range normalizedHeader {
				if h == expected {
					found = true
					break
				}
			}
			if !found {
				missing = append(missing, expected)
			}
		}
		return c.Status(fiber.StatusBadRequest).JSON(ErrorBody{
			Error:   "invalid_input",
			Message: fmt.Sprintf("CSV missing required columns: %s. Expected: first_name, middle_name, last_name, gender, date_of_birth", strings.Join(missing, ", ")),
		})
	}

	// Rewind the file to the beginning after header read
	_, _ = f.Seek(0, io.SeekStart)

	importID, err := h.svc.ImportCSV(c.Context(), tenantID, f)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(ErrorBody{
			Error:   "internal_error",
			Message: err.Error(),
		})
	}

	return c.Status(fiber.StatusAccepted).JSON(ImportResponse{
		ImportID: importID,
	})
}

// ImportStream handles GET /api/v1/students/import/stream?id=...
//
// @Summary      SSE stream for import progress
// @Description  Server-Sent Events endpoint that streams import progress updates.
// @Tags         Students
// @Produce      text/event-stream
// @Param        id   query  string  true  "Import ID from POST /import"
// @Success      200  {object}  string  "SSE event stream"
// @Failure      404  {object}  ErrorBody  "Import not found"
// @Router       /api/v1/students/import/stream [get]
func (h *Handler) ImportStream(c *fiber.Ctx) error {
	importID := c.Query("id")
	if importID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(ErrorBody{
			Error:   "invalid_input",
			Message: "import id is required",
		})
	}

	ch := h.svc.GetImportStream(importID)
	if ch == nil {
		return c.Status(fiber.StatusNotFound).JSON(ErrorBody{
			Error:   "not_found",
			Message: "import not found or already completed",
		})
	}

	done := h.svc.GetImportDone(importID)

	// Set SSE headers
	c.Set("Content-Type", "text/event-stream")
	c.Set("Cache-Control", "no-cache")
	c.Set("Connection", "keep-alive")
	c.Set("X-Accel-Buffering", "no")

	c.Context().SetBodyStreamWriter(func(w *bufio.Writer) {
		defer func() {
			// Send termination event
			_, _ = fmt.Fprintf(w, "event: done\ndata: {}\n\n")
			_ = w.Flush()
		}()

		for {
			select {
			case progress, ok := <-ch:
				if !ok {
					return
				}
				// Format as SSE data event
				line := fmt.Sprintf(`data: {"status":"%s"`, progress.Status)
				if progress.Total > 0 {
					line += fmt.Sprintf(`,"total":%d`, progress.Total)
				}
				if progress.Current > 0 {
					line += fmt.Sprintf(`,"current":%d`, progress.Current)
				}
				if progress.Success > 0 {
					line += fmt.Sprintf(`,"success":%d`, progress.Success)
				}
				if progress.Failed > 0 {
					line += fmt.Sprintf(`,"failed":%d`, progress.Failed)
				}
				if progress.Error != "" {
					line += fmt.Sprintf(`,"download_url":"%s"`, progress.Error)
				}
				line += "}\n\n"

				_, _ = fmt.Fprint(w, line)
				_ = w.Flush()

				// If terminal event, close the channel reader
				if progress.Status == "completed" || progress.Status == "error" {
					return
				}

			case <-done:
				return

			case <-c.Context().Done():
				return
			}
		}
	})

	return nil
}

// ImportErrors handles GET /api/v1/students/import/errors?id=...
//
// @Summary      Download error CSV
// @Description  Returns a CSV file containing the rows that failed validation during import.
// @Tags         Students
// @Produce      text/csv
// @Param        id   query  string  true  "Error log ID"
// @Success      200  {string}  string  "CSV file with error rows"
// @Failure      404  {object}  ErrorBody  "Error log not found"
// @Router       /api/v1/students/import/errors [get]
func (h *Handler) ImportErrors(c *fiber.Ctx) error {
	errorID := c.Query("id")
	if errorID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(ErrorBody{
			Error:   "invalid_input",
			Message: "error id is required",
		})
	}

	csvData, err := h.svc.GetErrorCSV(c.Context(), errorID)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(ErrorBody{
			Error:   "not_found",
			Message: "error log not found or expired",
		})
	}

	c.Set("Content-Type", "text/csv")
	c.Set("Content-Disposition", fmt.Sprintf(`attachment; filename="import_errors_%s.csv"`, errorID))
	return c.SendString(csvData)
}

// Module is an fx-compatible module for the students domain.
var Module = fx.Module("students",
	fx.Provide(
		NewRepository,
		NewService,
		NewHandler,
	),
)
