package routes

import (
	c "go-fiber-test/controllers"

	"github.com/gofiber/fiber/v2"
)

func Routes(app *fiber.App) {
	product := app.Group("/product")
	product.Get("/", c.GetProducts)
	product.Get("/:id", c.GetProduct)
	product.Post("/", c.AddProduct)
	product.Put("/:id", c.UpdateProduct)
	product.Delete("/:id", c.RemoveProduct)

	order := app.Group("/order")
	order.Get("/", c.GetOrders)
	order.Get("/:id", c.GetOrder)
	order.Post("/:id", c.AddOrder)
	order.Put("/:id", c.UpdateOrder)
	order.Delete("/:id", c.RemoveOrder)

	user := app.Group("/user")
	user.Post("/register", c.Register)
	user.Post("/login", c.Login)
}
