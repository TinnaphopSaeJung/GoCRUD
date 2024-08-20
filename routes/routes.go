package routes

import (
	c "go-fiber-test/controllers"
	md "go-fiber-test/middleware"

	"github.com/gofiber/fiber/v2"
)

func Routes(app *fiber.App) {
	product := app.Group("/product")
	product.Get("/", c.GetProducts)
	product.Get("/:product_id/image/:image_id", c.GetProductImage)
	product.Get("/:id", c.GetProduct)
	product.Post("/", md.AuthRequired, md.RoleRequired("admin"), c.AddProduct)
	product.Put("/:id", md.AuthRequired, md.RoleRequired("admin"), c.UpdateProduct)
	product.Delete("/:id", md.AuthRequired, md.RoleRequired("admin"), c.RemoveProduct)
	product.Delete("/:product_id/image/:image_id", md.AuthRequired, md.RoleRequired("admin"), c.RemoveImage)

	order := app.Group("/order")
	order.Get("/", md.AuthRequired, md.RoleRequired("admin"), c.GetOrders)
	order.Get("/:userId", md.AuthRequired, c.GetOrder)
	order.Post("/:id", md.AuthRequired, md.RoleRequired("user"), c.AddOrder)
	order.Put("/:id", md.AuthRequired, md.RoleRequired("user"), c.UpdateOrder)
	order.Delete("/:id", md.AuthRequired, md.RoleRequired("user"), c.RemoveOrder)

	user := app.Group("/user")
	user.Get("/", md.AuthRequired, md.RoleRequired("admin"), c.GetUsers)
	user.Post("/register", c.Register)
	user.Post("/login", c.Login)
	user.Post("/logout", c.Logout)
	user.Post("/refresh-token", c.RefreshToken)
	user.Put("/:id", md.AuthRequired, c.UpdateUser)
	user.Put("/restore/:id", md.AuthRequired, md.RoleRequired("admin"), c.RestoreUser)
	user.Put("/approve", md.AuthRequired, md.RoleRequired("admin"), c.Approve)
	user.Delete("/:id", md.AuthRequired, c.SoftDeleteUser)
	user.Delete("/bin/:id", md.AuthRequired, md.RoleRequired("admin"), c.HardDeleteUser)
}
