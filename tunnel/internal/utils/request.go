package utils

import (
	"github.com/gofiber/fiber/v2"
	"github.com/gookit/validate"
)

func BodyParser(c *fiber.Ctx, bindVar any) error {
	if err := c.BodyParser(bindVar); err != nil {
		return err
	}
	v := validate.Struct(bindVar)
	if !v.Validate() {
		return v.Errors
	}
	return nil
}

func ErrBadRequest(c *fiber.Ctx, message any) error {
	return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"message": message})
}

func ErrInternalServerError(c *fiber.Ctx, message any) error {
	return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"message": message})
}
