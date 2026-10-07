package http

import (
	"gin-boilerplate/internal/delivery/http/common"
	"gin-boilerplate/internal/delivery/http/dto"
	"gin-boilerplate/internal/delivery/http/response"
	"gin-boilerplate/internal/domain/shared"
	organizationdto "gin-boilerplate/internal/organizations/adapters/http/dto"
	organizationdomain "gin-boilerplate/internal/organizations/domain"
	"net/http"

	"github.com/gin-gonic/gin"
)

type OrganizationHandler struct {
	organizationUseCase organizationdomain.OrganizationUseCase
}

func NewOrganizationHandler(organizationUseCase organizationdomain.OrganizationUseCase) *OrganizationHandler {
	return &OrganizationHandler{
		organizationUseCase: organizationUseCase,
	}
}

// FindOrganization godoc
// @Summary      Get organization by ID
// @Description  Retrieve an organization by UUID. Requires 'view_organization' permission.
// @Tags         Organizations
// @Produce      json
// @Security     BearerAuth
// @Param        id   path      string  true  "Organization UUID" format(uuid)
// @Success      200  {object}  response.ApiResponse{data=organizationdto.OrganizationResponse} "Organization retrieved successfully"
// @Failure      401  {object}  response.ApiResponse "Unauthorized - Missing or invalid token"
// @Failure      403  {object}  response.ApiResponse "Forbidden - Requires view_organization permission"
// @Failure      404  {object}  response.ApiResponse "Not Found - Organization not found"
// @Failure      422  {object}  response.ApiResponse "Unprocessable Entity - Invalid UUID format"
// @Failure      500  {object}  response.ApiResponse "Internal server error"
// @Router       /api/v1/organizations/{id} [get]
func (h *OrganizationHandler) FindOrganization(c *gin.Context) {
	organizationID, err := common.ParamID(c, "id")
	if err != nil {
		_ = c.Error(err)
		return
	}

	organization, err := h.organizationUseCase.Find(c.Request.Context(), organizationID)
	if err != nil {
		_ = c.Error(err)
		return
	}

	response.Success(c, http.StatusOK, "Organization retrieved successfully", organizationdto.ToOrganizationResponse(organization))
}

// ListOrganizations godoc
// @Summary      List organizations
// @Description  Get a paginated list of organizations. Requires 'view_organization' permission.
// @Tags         Organizations
// @Produce      json
// @Security     BearerAuth
// @Param        page   query     int     false  "Page number (default: 1)" minimum(1)
// @Param        limit  query     int     false  "Items per page (default: 10, max: 100)" minimum(1) maximum(100)
// @Param        sort   query     string  false  "Sorting criteria (e.g. name:asc, created_at:desc or -created_at)"
// @Param        name   query     string  false  "Filter by organization name (partial match)"
// @Success      200    {object}  response.ApiResponse{data=dto.PaginatedResponse[organizationdto.OrganizationResponse]} "Organizations retrieved successfully"
// @Failure      401    {object}  response.ApiResponse "Unauthorized - Missing or invalid token"
// @Failure      403    {object}  response.ApiResponse "Forbidden - Requires view_organization permission"
// @Failure      422    {object}  response.ApiResponse "Unprocessable Entity - Invalid filter or sorting parameter"
// @Failure      500    {object}  response.ApiResponse "Internal server error"
// @Router       /api/v1/organizations [get]
func (h *OrganizationHandler) ListOrganizations(c *gin.Context) {
	params, err := common.PaginationParams(c)
	if err != nil {
		_ = c.Error(err)
		return
	}

	var filters []shared.Filter
	if name := c.Query("name"); name != "" {
		filters = append(filters, shared.Filter{
			Field:    "name",
			Operator: shared.OperatorLike,
			Value:    "%" + name + "%",
		})
	}

	result, err := h.organizationUseCase.List(c.Request.Context(), params, filters)
	if err != nil {
		_ = c.Error(err)
		return
	}

	resp := dto.ToPaginatedResponse(result, organizationdto.ToOrganizationResponse)
	response.Success(c, http.StatusOK, "Organizations retrieved successfully", resp)
}

// CreateOrganization godoc
// @Summary      Create an organization
// @Description  Create an organization. Requires 'create_organization' permission.
// @Tags         Organizations
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        request body organizationdto.CreateOrganizationRequest true "Organization details"
// @Success      201  {object}  response.ApiResponse{data=organizationdto.OrganizationResponse} "Organization created successfully"
// @Failure      401  {object}  response.ApiResponse "Unauthorized - Missing or invalid token"
// @Failure      403  {object}  response.ApiResponse "Forbidden - Requires create_organization permission"
// @Failure      409  {object}  response.ApiResponse "Conflict - Organization name already exists"
// @Failure      422  {object}  response.ApiResponse "Unprocessable Entity - Invalid request payload"
// @Failure      500  {object}  response.ApiResponse "Internal server error"
// @Router       /api/v1/organizations [post]
func (h *OrganizationHandler) CreateOrganization(c *gin.Context) {
	req, err := common.BindJSON[organizationdto.CreateOrganizationRequest](c)
	if err != nil {
		_ = c.Error(err)
		return
	}

	organization := &organizationdomain.Organization{
		Name: req.Name,
	}

	if err := h.organizationUseCase.Create(c.Request.Context(), organization); err != nil {
		_ = c.Error(err)
		return
	}

	response.Success(c, http.StatusCreated, "Organization created successfully", organizationdto.ToOrganizationResponse(organization))
}

// UpdateOrganization godoc
// @Summary      Update an organization
// @Description  Update an organization's name by UUID. Requires 'update_organization' permission.
// @Tags         Organizations
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id       path      string  true  "Organization UUID" format(uuid)
// @Param        request  body      organizationdto.UpdateOrganizationRequest true "Updated organization details"
// @Success      200      {object}  response.ApiResponse{data=organizationdto.OrganizationResponse} "Organization updated successfully"
// @Failure      401      {object}  response.ApiResponse "Unauthorized - Missing or invalid token"
// @Failure      403      {object}  response.ApiResponse "Forbidden - Requires update_organization permission"
// @Failure      404      {object}  response.ApiResponse "Not Found - Organization not found"
// @Failure      409      {object}  response.ApiResponse "Conflict - Organization name already exists"
// @Failure      422      {object}  response.ApiResponse "Unprocessable Entity - Invalid UUID format or request payload"
// @Failure      500      {object}  response.ApiResponse "Internal server error"
// @Router       /api/v1/organizations/{id} [put]
func (h *OrganizationHandler) UpdateOrganization(c *gin.Context) {
	organizationID, err := common.ParamID(c, "id")
	if err != nil {
		_ = c.Error(err)
		return
	}

	req, err := common.BindJSON[organizationdto.UpdateOrganizationRequest](c)
	if err != nil {
		_ = c.Error(err)
		return
	}

	organization := &organizationdomain.Organization{
		ID:   organizationID,
		Name: req.Name,
	}

	if err := h.organizationUseCase.Update(c.Request.Context(), organization); err != nil {
		_ = c.Error(err)
		return
	}

	response.Success(c, http.StatusOK, "Organization updated successfully", organizationdto.ToOrganizationResponse(organization))
}

// DeleteOrganization godoc
// @Summary      Delete an organization
// @Description  Soft delete an organization by UUID. Requires 'delete_organization' permission.
// @Tags         Organizations
// @Produce      json
// @Security     BearerAuth
// @Param        id   path      string  true  "Organization UUID" format(uuid)
// @Success      204  {object}  nil "Organization deleted successfully"
// @Failure      401  {object}  response.ApiResponse "Unauthorized - Missing or invalid token"
// @Failure      403  {object}  response.ApiResponse "Forbidden - Requires delete_organization permission"
// @Failure      404  {object}  response.ApiResponse "Not Found - Organization not found"
// @Failure      422  {object}  response.ApiResponse "Unprocessable Entity - Invalid UUID format"
// @Failure      500  {object}  response.ApiResponse "Internal server error"
// @Router       /api/v1/organizations/{id} [delete]
func (h *OrganizationHandler) DeleteOrganization(c *gin.Context) {
	organizationID, err := common.ParamID(c, "id")
	if err != nil {
		_ = c.Error(err)
		return
	}

	if err := h.organizationUseCase.Delete(c.Request.Context(), organizationID); err != nil {
		_ = c.Error(err)
		return
	}

	c.Status(http.StatusNoContent)
}
