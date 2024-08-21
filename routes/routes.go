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
	product.Get("/:productId", c.GetProduct)
	product.Post("/", md.AuthRequired, md.RoleRequired("admin"), c.AddProduct)
	product.Put("/:productId", md.AuthRequired, md.RoleRequired("admin"), c.UpdateProduct)
	product.Put("/restore/:productId", md.AuthRequired, md.RoleRequired("admin"), c.RestoreProduct)
	product.Delete("/:productId", md.AuthRequired, md.RoleRequired("admin"), c.SoftDeleteProduct)
	product.Delete("/bin/:productId", md.AuthRequired, md.RoleRequired("admin"), c.HardDeleteProduct)
	product.Delete("/:product_id/image/:image_id", md.AuthRequired, md.RoleRequired("admin"), c.RemoveImage)

	order := app.Group("/order")
	order.Get("/", md.AuthRequired, md.RoleRequired("admin"), c.GetOrders)
	order.Get("/:userId", md.AuthRequired, c.GetOrder)
	order.Post("/:userId", md.AuthRequired, md.RoleRequired("user"), c.AddOrder)
	order.Put("/:orderId", md.AuthRequired, md.RoleRequired("user"), c.UpdateOrder)
	order.Delete("/:orderId", md.AuthRequired, md.RoleRequired("user"), c.RemoveOrder)

	user := app.Group("/user")
	user.Get("/", md.AuthRequired, md.RoleRequired("admin"), c.GetUsers)
	user.Post("/register", c.Register)
	user.Post("/login", c.Login)
	user.Post("/logout", c.Logout)
	user.Post("/refresh-token", c.RefreshToken)
	user.Put("/approve", md.AuthRequired, md.RoleRequired("admin"), c.Approve)
	user.Put("/:userId", md.AuthRequired, c.UpdateUser)
	user.Put("/restore/:userId", md.AuthRequired, md.RoleRequired("admin"), c.RestoreUser)
	user.Delete("/:userId", md.AuthRequired, c.SoftDeleteUser)
	user.Delete("/bin/:userId", md.AuthRequired, md.RoleRequired("admin"), c.HardDeleteUser)
}
