package rest

import (
	"flyup/internal/helper"
	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/schema"
	"mime"
)

func BindBody(ctx fiber.Ctx, out any) error {
	contentType, _, _ := mime.ParseMediaType(ctx.Get("Content-Type"))
	if contentType == fiber.MIMEApplicationForm || contentType == fiber.MIMEMultipartForm {
		values := make(map[string][]string)
		if contentType == fiber.MIMEMultipartForm {
			form, err := ctx.MultipartForm()
			if err != nil {
				return err
			}
			values = form.Value
		} else {
			ctx.Request().PostArgs().VisitAll(func(key, value []byte) { values[string(key)] = append(values[string(key)], string(value)) })
		}
		// Request structs use JSON field names for both JSON and HTML forms.
		decoder := schema.NewDecoder()
		decoder.SetAliasTag("json")
		if err := decoder.Decode(out, values); err != nil {
			return err
		}
		return helper.ValidateInputLimits(out)
	}
	if err := ctx.Bind().Body(out); err != nil {
		return err
	}
	return helper.ValidateInputLimits(out)
}

func BindJSON(ctx fiber.Ctx, out any) error {
	if err := ctx.Bind().JSON(out); err != nil {
		return err
	}
	return helper.ValidateInputLimits(out)
}

func BindQuery(ctx fiber.Ctx, out any) error {
	if err := ctx.Bind().Query(out); err != nil {
		return err
	}
	return helper.ValidateInputLimits(out)
}
