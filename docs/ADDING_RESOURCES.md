# Adding New Resources Guide

This guide walks you through adding a new resource (e.g., "Products") to your API.

## Overview

To add a new resource, you'll need to:
1. Define the model
2. Add repository methods
3. Implement business logic in service layer
4. Create HTTP handlers
5. Register routes
6. Add tests (optional)

## Example: Adding a "Products" Resource

Let's add a complete CRUD API for products.

### Step 1: Define the Model

Add to `internal/models/models.go`:

```go
// Product represents a product in the system
type Product struct {
    ID          string    `json:"id" gorm:"primaryKey" firestore:"id"`
    Name        string    `json:"name" gorm:"not null" firestore:"name" binding:"required"`
    Description string    `json:"description" gorm:"type:text" firestore:"description"`
    Price       float64   `json:"price" gorm:"not null" firestore:"price" binding:"required,gt=0"`
    Stock       int       `json:"stock" gorm:"not null;default:0" firestore:"stock" binding:"omitempty,gte=0"`
    Category    string    `json:"category" gorm:"not null" firestore:"category" binding:"required"`
    UserID      string    `json:"user_id" gorm:"not null;index" firestore:"user_id"`
    CreatedAt   time.Time `json:"created_at" gorm:"autoCreateTime" firestore:"created_at"`
    UpdatedAt   time.Time `json:"updated_at" gorm:"autoUpdateTime" firestore:"updated_at"`
}

// ProductCreateRequest represents product creation data
type ProductCreateRequest struct {
    Name        string  `json:"name" binding:"required"`
    Description string  `json:"description"`
    Price       float64 `json:"price" binding:"required,gt=0"`
    Stock       int     `json:"stock" binding:"omitempty,gte=0"`
    Category    string  `json:"category" binding:"required"`
}

// ProductUpdateRequest represents product update data
type ProductUpdateRequest struct {
    Name        string  `json:"name" binding:"omitempty"`
    Description string  `json:"description" binding:"omitempty"`
    Price       float64 `json:"price" binding:"omitempty,gt=0"`
    Stock       int     `json:"stock" binding:"omitempty,gte=0"`
    Category    string  `json:"category" binding:"omitempty"`
}
```

### Step 2: Add Repository Interface

Add to `internal/repository/repository.go`:

```go
// Product operations
CreateProduct(ctx context.Context, product *models.Product) error
GetProductByID(ctx context.Context, id string) (*models.Product, error)
GetProducts(ctx context.Context, userID string, pagination *models.PaginationQuery) ([]*models.Product, int64, error)
GetProductsByCategory(ctx context.Context, userID, category string, pagination *models.PaginationQuery) ([]*models.Product, int64, error)
UpdateProduct(ctx context.Context, product *models.Product) error
DeleteProduct(ctx context.Context, id string) error
```

### Step 3: Implement MySQL Repository

Add to `internal/repository/mysql_repository.go`:

```go
// Product operations
func (r *MySQLRepository) CreateProduct(ctx context.Context, product *models.Product) error {
    product.ID = uuid.New().String()
    product.CreatedAt = time.Now()
    product.UpdatedAt = time.Now()
    return r.db.WithContext(ctx).Create(product).Error
}

func (r *MySQLRepository) GetProductByID(ctx context.Context, id string) (*models.Product, error) {
    var product models.Product
    err := r.db.WithContext(ctx).Where("id = ?", id).First(&product).Error
    if err != nil {
        if err == gorm.ErrRecordNotFound {
            return nil, fmt.Errorf("product not found")
        }
        return nil, err
    }
    return &product, nil
}

func (r *MySQLRepository) GetProducts(ctx context.Context, userID string, pagination *models.PaginationQuery) ([]*models.Product, int64, error) {
    var products []*models.Product
    var total int64

    query := r.db.WithContext(ctx).Model(&models.Product{})
    
    if userID != "" {
        query = query.Where("user_id = ?", userID)
    }

    // Get total count
    if err := query.Count(&total).Error; err != nil {
        return nil, 0, err
    }

    // Apply pagination
    offset := (pagination.Page - 1) * pagination.PageSize
    orderClause := fmt.Sprintf("%s %s", pagination.SortBy, pagination.Order)
    
    err := query.
        Order(orderClause).
        Limit(pagination.PageSize).
        Offset(offset).
        Find(&products).Error

    return products, total, err
}

func (r *MySQLRepository) GetProductsByCategory(ctx context.Context, userID, category string, pagination *models.PaginationQuery) ([]*models.Product, int64, error) {
    var products []*models.Product
    var total int64

    query := r.db.WithContext(ctx).Model(&models.Product{}).Where("category = ?", category)
    
    if userID != "" {
        query = query.Where("user_id = ?", userID)
    }

    if err := query.Count(&total).Error; err != nil {
        return nil, 0, err
    }

    offset := (pagination.Page - 1) * pagination.PageSize
    orderClause := fmt.Sprintf("%s %s", pagination.SortBy, pagination.Order)
    
    err := query.
        Order(orderClause).
        Limit(pagination.PageSize).
        Offset(offset).
        Find(&products).Error

    return products, total, err
}

func (r *MySQLRepository) UpdateProduct(ctx context.Context, product *models.Product) error {
    product.UpdatedAt = time.Now()
    return r.db.WithContext(ctx).Save(product).Error
}

func (r *MySQLRepository) DeleteProduct(ctx context.Context, id string) error {
    return r.db.WithContext(ctx).Delete(&models.Product{}, "id = ?", id).Error
}
```

### Step 4: Implement Firebase Repository

Add to `internal/repository/firebase_repository.go`:

```go
// Product operations
func (r *FirebaseRepository) CreateProduct(ctx context.Context, product *models.Product) error {
    product.ID = uuid.New().String()
    product.CreatedAt = time.Now()
    product.UpdatedAt = time.Now()

    _, err := r.client.Collection("products").Doc(product.ID).Set(ctx, product)
    return err
}

func (r *FirebaseRepository) GetProductByID(ctx context.Context, id string) (*models.Product, error) {
    doc, err := r.client.Collection("products").Doc(id).Get(ctx)
    if err != nil {
        return nil, fmt.Errorf("product not found: %w", err)
    }

    var product models.Product
    if err := doc.DataTo(&product); err != nil {
        return nil, err
    }

    return &product, nil
}

func (r *FirebaseRepository) GetProducts(ctx context.Context, userID string, pagination *models.PaginationQuery) ([]*models.Product, int64, error) {
    query := r.client.Collection("products").Query

    if userID != "" {
        query = query.Where("user_id", "==", userID)
    }

    countSnap, err := query.Documents(ctx).GetAll()
    if err != nil {
        return nil, 0, err
    }
    total := int64(len(countSnap))

    var direction firestore.Direction
    if pagination.Order == "desc" {
        direction = firestore.Desc
    } else {
        direction = firestore.Asc
    }
    query = query.OrderBy(pagination.SortBy, direction)

    offset := (pagination.Page - 1) * pagination.PageSize
    query = query.Offset(offset).Limit(pagination.PageSize)

    iter := query.Documents(ctx)
    docs, err := iter.GetAll()
    if err != nil {
        return nil, 0, err
    }

    products := make([]*models.Product, 0, len(docs))
    for _, doc := range docs {
        var product models.Product
        if err := doc.DataTo(&product); err != nil {
            return nil, 0, err
        }
        products = append(products, &product)
    }

    return products, total, nil
}

func (r *FirebaseRepository) GetProductsByCategory(ctx context.Context, userID, category string, pagination *models.PaginationQuery) ([]*models.Product, int64, error) {
    query := r.client.Collection("products").Where("category", "==", category).Query

    if userID != "" {
        query = query.Where("user_id", "==", userID)
    }

    // Similar implementation as GetProducts...
    // (code omitted for brevity)
}

func (r *FirebaseRepository) UpdateProduct(ctx context.Context, product *models.Product) error {
    product.UpdatedAt = time.Now()
    _, err := r.client.Collection("products").Doc(product.ID).Set(ctx, product)
    return err
}

func (r *FirebaseRepository) DeleteProduct(ctx context.Context, id string) error {
    _, err := r.client.Collection("products").Doc(id).Delete(ctx)
    return err
}
```

### Step 5: Add Service Layer

Add to `internal/service/service.go`:

```go
// Product operations
func (s *Service) CreateProduct(ctx context.Context, userID string, req *models.ProductCreateRequest) (*models.Product, error) {
    product := &models.Product{
        Name:        req.Name,
        Description: req.Description,
        Price:       req.Price,
        Stock:       req.Stock,
        Category:    req.Category,
        UserID:      userID,
    }

    if err := s.repo.CreateProduct(ctx, product); err != nil {
        return nil, fmt.Errorf("failed to create product: %w", err)
    }

    logger.Info("Product created successfully", "product_id", product.ID, "user_id", userID)
    return product, nil
}

func (s *Service) GetProductByID(ctx context.Context, productID, userID string) (*models.Product, error) {
    product, err := s.repo.GetProductByID(ctx, productID)
    if err != nil {
        return nil, err
    }

    // Check ownership
    if product.UserID != userID {
        return nil, errors.New("unauthorized access to product")
    }

    return product, nil
}

func (s *Service) GetProducts(ctx context.Context, userID string, pagination *models.PaginationQuery) (*models.PaginatedResponse, error) {
    products, total, err := s.repo.GetProducts(ctx, userID, pagination)
    if err != nil {
        return nil, fmt.Errorf("failed to get products: %w", err)
    }

    totalPages := int(total) / pagination.PageSize
    if int(total)%pagination.PageSize > 0 {
        totalPages++
    }

    return &models.PaginatedResponse{
        Data:       products,
        Page:       pagination.Page,
        PageSize:   pagination.PageSize,
        TotalItems: total,
        TotalPages: totalPages,
    }, nil
}

func (s *Service) GetProductsByCategory(ctx context.Context, userID, category string, pagination *models.PaginationQuery) (*models.PaginatedResponse, error) {
    products, total, err := s.repo.GetProductsByCategory(ctx, userID, category, pagination)
    if err != nil {
        return nil, fmt.Errorf("failed to get products: %w", err)
    }

    totalPages := int(total) / pagination.PageSize
    if int(total)%pagination.PageSize > 0 {
        totalPages++
    }

    return &models.PaginatedResponse{
        Data:       products,
        Page:       pagination.Page,
        PageSize:   pagination.PageSize,
        TotalItems: total,
        TotalPages: totalPages,
    }, nil
}

func (s *Service) UpdateProduct(ctx context.Context, productID, userID string, req *models.ProductUpdateRequest) (*models.Product, error) {
    product, err := s.repo.GetProductByID(ctx, productID)
    if err != nil {
        return nil, err
    }

    // Check ownership
    if product.UserID != userID {
        return nil, errors.New("unauthorized access to product")
    }

    // Update fields
    if req.Name != "" {
        product.Name = req.Name
    }
    if req.Description != "" {
        product.Description = req.Description
    }
    if req.Price > 0 {
        product.Price = req.Price
    }
    if req.Stock >= 0 {
        product.Stock = req.Stock
    }
    if req.Category != "" {
        product.Category = req.Category
    }

    if err := s.repo.UpdateProduct(ctx, product); err != nil {
        return nil, fmt.Errorf("failed to update product: %w", err)
    }

    logger.Info("Product updated successfully", "product_id", productID, "user_id", userID)
    return product, nil
}

func (s *Service) DeleteProduct(ctx context.Context, productID, userID string) error {
    product, err := s.repo.GetProductByID(ctx, productID)
    if err != nil {
        return err
    }

    // Check ownership
    if product.UserID != userID {
        return errors.New("unauthorized access to product")
    }

    if err := s.repo.DeleteProduct(ctx, productID); err != nil {
        return fmt.Errorf("failed to delete product: %w", err)
    }

    logger.Info("Product deleted successfully", "product_id", productID, "user_id", userID)
    return nil
}
```

### Step 6: Add HTTP Handlers

Add to `internal/handlers/handlers.go`:

```go
// CreateProduct godoc
// @Summary Create a new product
// @Description Create a new product for the authenticated user
// @Tags products
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body models.ProductCreateRequest true "Product details"
// @Success 201 {object} models.SuccessResponse{data=models.Product}
// @Failure 400 {object} models.ErrorResponse
// @Failure 401 {object} models.ErrorResponse
// @Router /api/v1/products [post]
func (h *Handler) CreateProduct(c *gin.Context) {
    userID := middleware.GetUserID(c)

    var req models.ProductCreateRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(http.StatusBadRequest, models.ErrorResponse{
            Error:   "invalid_request",
            Message: err.Error(),
        })
        return
    }

    product, err := h.service.CreateProduct(c.Request.Context(), userID, &req)
    if err != nil {
        c.JSON(http.StatusInternalServerError, models.ErrorResponse{
            Error:   "creation_failed",
            Message: err.Error(),
        })
        return
    }

    c.JSON(http.StatusCreated, models.SuccessResponse{
        Message: "Product created successfully",
        Data:    product,
    })
}

// GetProducts godoc
// @Summary Get all products
// @Description Get all products for the authenticated user with pagination
// @Tags products
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param page query int false "Page number" default(1)
// @Param page_size query int false "Page size" default(10)
// @Param category query string false "Filter by category"
// @Success 200 {object} models.PaginatedResponse
// @Failure 401 {object} models.ErrorResponse
// @Router /api/v1/products [get]
func (h *Handler) GetProducts(c *gin.Context) {
    userID := middleware.GetUserID(c)
    category := c.Query("category")

    var pagination models.PaginationQuery
    if err := c.ShouldBindQuery(&pagination); err != nil {
        c.JSON(http.StatusBadRequest, models.ErrorResponse{
            Error:   "invalid_request",
            Message: err.Error(),
        })
        return
    }

    // Set defaults
    if pagination.Page == 0 {
        pagination.Page = 1
    }
    if pagination.PageSize == 0 {
        pagination.PageSize = 10
    }
    if pagination.SortBy == "" {
        pagination.SortBy = "created_at"
    }
    if pagination.Order == "" {
        pagination.Order = "desc"
    }

    var result *models.PaginatedResponse
    var err error

    if category != "" {
        result, err = h.service.GetProductsByCategory(c.Request.Context(), userID, category, &pagination)
    } else {
        result, err = h.service.GetProducts(c.Request.Context(), userID, &pagination)
    }

    if err != nil {
        c.JSON(http.StatusInternalServerError, models.ErrorResponse{
            Error:   "retrieval_failed",
            Message: err.Error(),
        })
        return
    }

    c.JSON(http.StatusOK, result)
}

// GetProductByID, UpdateProduct, DeleteProduct follow similar patterns...
```

### Step 7: Register Routes

Add to `cmd/api/main.go` in the `setupRouter` function:

```go
// Product routes
products := protected.Group("/products")
{
    products.GET("", h.GetProducts)
    products.GET("/:id", h.GetProductByID)
    products.POST("", h.CreateProduct)
    products.PUT("/:id", h.UpdateProduct)
    products.DELETE("/:id", h.DeleteProduct)
}
```

### Step 8: Create Database Migration

Create `migrations/000002_create_products_table.up.sql`:

```sql
CREATE TABLE IF NOT EXISTS products (
    id VARCHAR(36) PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    price DECIMAL(10, 2) NOT NULL,
    stock INT NOT NULL DEFAULT 0,
    category VARCHAR(100) NOT NULL,
    user_id VARCHAR(36) NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    INDEX idx_user_id (user_id),
    INDEX idx_category (category),
    INDEX idx_created_at (created_at),
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
```

Create `migrations/000002_create_products_table.down.sql`:

```sql
DROP TABLE IF EXISTS products;
```

### Step 9: Generate Swagger Documentation

```bash
# Add Swagger comments to handlers (shown above)

# Generate docs
make swagger
```

### Step 10: Test Your New Resource

```bash
# Register and login
curl -X POST http://localhost:8080/api/v1/auth/register \
  -H "Content-Type: application/json" \
  -d '{"email": "test@example.com", "password": "password123", "name": "Test User"}'

# Login to get token
curl -X POST http://localhost:8080/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email": "test@example.com", "password": "password123"}'

# Create product
curl -X POST http://localhost:8080/api/v1/products \
  -H "Authorization: Bearer YOUR_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Laptop",
    "description": "High-performance laptop",
    "price": 999.99,
    "stock": 10,
    "category": "Electronics"
  }'

# Get all products
curl http://localhost:8080/api/v1/products \
  -H "Authorization: Bearer YOUR_TOKEN"

# Get products by category
curl "http://localhost:8080/api/v1/products?category=Electronics" \
  -H "Authorization: Bearer YOUR_TOKEN"
```

## Best Practices

1. **Validation**: Always validate input data using binding tags
2. **Authorization**: Check ownership before allowing modifications
3. **Logging**: Log important events (create, update, delete)
4. **Error Handling**: Provide clear error messages
5. **Pagination**: Support pagination for list endpoints
6. **Testing**: Write unit tests for each layer
7. **Documentation**: Add Swagger comments for all endpoints

## Common Patterns

### Search/Filter

Add filter support:

```go
func (r *MySQLRepository) SearchProducts(ctx context.Context, userID, searchTerm string) ([]*models.Product, error) {
    var products []*models.Product
    query := r.db.WithContext(ctx).Where("user_id = ?", userID)
    
    if searchTerm != "" {
        query = query.Where("name LIKE ? OR description LIKE ?", 
            "%"+searchTerm+"%", "%"+searchTerm+"%")
    }
    
    err := query.Find(&products).Error
    return products, err
}
```

### Soft Delete

Add soft delete support:

```go
type Product struct {
    // ... other fields
    DeletedAt gorm.DeletedAt `json:"-" gorm:"index"`
}
```

### Relations

Add related resources:

```go
type Product struct {
    // ... other fields
    Reviews []Review `json:"reviews,omitempty" gorm:"foreignKey:ProductID"`
}
```

## Need Help?

- Check existing resources (Album) for reference
- Review the [main README](../README.md)
- Open an issue if you get stuck

Happy coding! 🚀
