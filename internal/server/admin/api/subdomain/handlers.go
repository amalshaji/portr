package subdomain

import (
	"errors"
	"fmt"

	"github.com/amalshaji/portr/internal/server/admin/middleware"
	"github.com/amalshaji/portr/internal/server/admin/services"
	serverConfig "github.com/amalshaji/portr/internal/server/config"
	"github.com/amalshaji/portr/internal/utils"
	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

const invalidSubdomainMessage = "Use 1-63 lowercase letters, numbers, or internal hyphens"

type Handler struct {
	service *services.SubdomainService
	config  *serverConfig.AdminConfig
}

type createInput struct {
	Subdomain string `json:"subdomain"`
}

type reservationResponse struct {
	Subdomain   string                        `json:"subdomain"`
	CreatedAt   string                        `json:"created_at"`
	ClaimStatus services.SubdomainClaimStatus `json:"claim_status"`
}

func NewHandler(db *gorm.DB, config *serverConfig.AdminConfig) *Handler {
	return &Handler{service: services.NewSubdomainService(db), config: config}
}

func (h *Handler) List(c *fiber.Ctx) error {
	teamUser := middleware.GetCurrentTeamUser(c)
	if teamUser == nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Team context required"})
	}

	reservations, err := h.service.List(c.UserContext(), teamUser.ID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"code":    "reservation_load_failed",
			"message": "Failed to load reserved subdomains",
		})
	}

	data := make([]reservationResponse, 0, len(reservations))
	for _, item := range reservations {
		data = append(data, responseFor(item))
	}

	return c.JSON(fiber.Map{
		"data":        data,
		"count":       len(data),
		"limit":       h.config.ReservedSubdomainLimit,
		"base_domain": h.config.TunnelDomain,
	})
}

func (h *Handler) Create(c *fiber.Ctx) error {
	teamUser := middleware.GetCurrentTeamUser(c)
	if teamUser == nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Team context required"})
	}

	var input createInput
	if err := c.BodyParser(&input); err != nil {
		return apiError(c, fiber.StatusBadRequest, "invalid_input", "Invalid input")
	}

	subdomain := utils.NormalizeSubdomain(input.Subdomain)
	if err := utils.ValidateSubdomain(subdomain); err != nil {
		return apiError(c, fiber.StatusBadRequest, "invalid_subdomain", invalidSubdomainMessage)
	}

	reservation, err := h.service.Reserve(c.UserContext(), teamUser.ID, subdomain, h.config.ReservedSubdomainLimit)
	if err != nil {
		return h.handleServiceError(c, err)
	}

	return c.Status(fiber.StatusCreated).JSON(responseFor(*reservation))
}

func (h *Handler) Delete(c *fiber.Ctx) error {
	teamUser := middleware.GetCurrentTeamUser(c)
	if teamUser == nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Team context required"})
	}

	subdomain := utils.NormalizeSubdomain(c.Params("subdomain"))
	if err := utils.ValidateSubdomain(subdomain); err != nil {
		return apiError(c, fiber.StatusBadRequest, "invalid_subdomain", invalidSubdomainMessage)
	}

	if err := h.service.Release(c.UserContext(), teamUser.ID, subdomain); err != nil {
		return h.handleServiceError(c, err)
	}
	return c.SendStatus(fiber.StatusNoContent)
}

func (h *Handler) handleServiceError(c *fiber.Ctx, err error) error {
	switch {
	case errors.Is(err, services.ErrReservationExists):
		return apiError(c, fiber.StatusConflict, "already_reserved", "You already reserved this subdomain")
	case errors.Is(err, services.ErrReservationLimit):
		return apiError(c, fiber.StatusConflict, "reservation_limit_reached", fmt.Sprintf("You can reserve up to %d subdomains", h.config.ReservedSubdomainLimit))
	case errors.Is(err, services.ErrSubdomainUnavailable):
		return apiError(c, fiber.StatusConflict, "subdomain_unavailable", "This subdomain is unavailable")
	case errors.Is(err, services.ErrReservationNotFound):
		return apiError(c, fiber.StatusNotFound, "reservation_not_found", "Reserved subdomain not found")
	case errors.Is(err, services.ErrReservationUnavailable):
		return apiError(c, fiber.StatusServiceUnavailable, "reservation_busy", "Reservations are busy; try again")
	default:
		return apiError(c, fiber.StatusInternalServerError, "reservation_failed", "Failed to update reserved subdomains")
	}
}

func responseFor(reservation services.ReservedSubdomain) reservationResponse {
	return reservationResponse{
		Subdomain:   reservation.Reservation.Subdomain,
		CreatedAt:   reservation.Reservation.CreatedAt.UTC().Format("2006-01-02T15:04:05Z"),
		ClaimStatus: reservation.ClaimStatus,
	}
}

func apiError(c *fiber.Ctx, status int, code, message string) error {
	return c.Status(status).JSON(fiber.Map{"code": code, "message": message})
}
