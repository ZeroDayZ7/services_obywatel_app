package handler

import "github.com/zerodayz7/platform/services/auth-service/internal/features/users/service"

type UserHandler struct {
	userService *service.UserService
}

func NewUserHandler(userService *service.UserService) *UserHandler {
	return &UserHandler{userService: userService}
}
