package routes

import (
	c "go-fiber-test/controllers"
	md "go-fiber-test/middleware"

	"github.com/gofiber/fiber/v2"
)

func Routes(app *fiber.App) {
	product := app.Group("/product")
	product.Get("/", c.GetProducts)
	product.Get("/:id", md.AuthRequired, c.GetProduct)
	product.Post("/", md.AuthRequired, c.AddProduct)
	product.Put("/:id", md.AuthRequired, c.UpdateProduct)
	product.Delete("/:id", md.AuthRequired, c.RemoveProduct)
	product.Delete("/:product_id/image/:image_id", md.AuthRequired, c.RemoveImage)

	order := app.Group("/order")
	order.Get("/", md.AuthRequired, c.GetOrders)
	order.Get("/:id", md.AuthRequired, c.GetOrder)
	order.Post("/:id", md.AuthRequired, c.AddOrder)
	order.Put("/:id", md.AuthRequired, c.UpdateOrder)
	order.Delete("/:id", md.AuthRequired, c.RemoveOrder)

	user := app.Group("/user")
	user.Post("/register", c.Register)
	user.Post("/login", c.Login)
	user.Post("/refresh-token", c.RefreshToken)
}
