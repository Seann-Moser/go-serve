package generators

import (
	"fmt"
	"net/http"
)
// HandlerFuncs godoc
// @Summary empty
// @Tags account,GET
// @ID account_user-9df6dae28a065c2087fbd4eac002c2cd9de221e7
// @Description empty
// @Produce json 
// @Param account_id path string true "description" 
// @Param user_id path string true "description" 
// @Success 200 {object} response.BaseResponse{data=clientpkg.RequestData} "returning object"  
// @Failure 400 {object} response.BaseResponse "invalid request to endpoint"
// @Failure 500 {object} response.BaseResponse "failed"
// @Failure 401 {object} response.BaseResponse "unauthorized request to endpoint"
// @Router /account/{account_id}/user/{user_id} [GET] 
func HandlerFuncs(w http.ResponseWriter, r *http.Request) {
	// Set the content type to plain text
	w.Header().Set("Content-Type", "text/plain")

	// Respond with "pong"
	fmt.Fprintln(w, "pong")
}