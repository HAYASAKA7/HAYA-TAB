package app

import (
	"fmt"
	"haya-tab/pkg/store"
	"time"
)

// GetCategories returns the list of categories
func (a *App) GetCategories() []store.Category {
	categories, err := a.store.GetCategories()
	if err != nil {
		a.logger.Error("Error getting categories: %v", err)
		return []store.Category{}
	}
	return categories
}

// GetRecentCategories returns the list of recently accessed categories
func (a *App) GetRecentCategories(limit int) []store.Category {
	categories, err := a.store.GetRecentCategories(limit)
	if err != nil {
		a.logger.Error("Error getting recent categories: %v", err)
		return []store.Category{}
	}
	return categories
}

// AddCategory adds a new category
func (a *App) AddCategory(cat store.Category) error {
	// Generate ID if missing (though frontend might handle it, safer here or ensure uniqueness)
	if cat.ID == "" {
		cat.ID = fmt.Sprintf("cat_%d", time.Now().UnixNano())
	}
	return a.store.AddCategory(cat)
}

// DeleteCategory deletes a category
func (a *App) DeleteCategory(id string) error {
	return a.store.DeleteCategory(id)
}

// MoveCategory moves a category into another category
func (a *App) MoveCategory(id, newParentID string) error {
	if id == newParentID {
		return fmt.Errorf("cannot move category into itself")
	}
	// Note: A robust implementation should also check for circular dependency
	return a.store.MoveCategory(id, newParentID)
}
