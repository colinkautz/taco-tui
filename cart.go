package main

import "fmt"

type CartItem struct {
	Item     MenuItem
	Quantity int
}

type Cart struct {
	items map[string]*CartItem
	order []string
}

func NewCart() *Cart {
	return &Cart{
		items: make(map[string]*CartItem),
	}
}

func (c *Cart) AddItem(item MenuItem) {
	if cartItem, ok := c.items[item.ProductID]; ok {
		cartItem.Quantity++
	} else {
		c.items[item.ProductID] = &CartItem{
			Item:     item,
			Quantity: 1,
		}

		c.order = append(c.order, item.ProductID)
	}
}

func (c *Cart) RemoveItem(productID string) {
	cartItem, ok := c.items[productID]

	if !ok {
		return
	}

	cartItem.Quantity--

	if cartItem.Quantity <= 0 {
		delete(c.items, productID)
		for i, j := range c.order {
			if j == productID {
				c.order = append(c.order[:i], c.order[i+1:]...)
				break
			}
		}
	}
}

func (c *Cart) Items() []*CartItem {
	result := make([]*CartItem, 0, len(c.order))

	for _, productID := range c.order {
		if entry, ok := c.items[productID]; ok {
			result = append(result, entry)
		}
	}

	return result
}

func (c *Cart) Total() float64 {
	var total float64

	for _, item := range c.items {
		total += item.Item.Price * float64(item.Quantity)
	}

	return total
}

func (c *Cart) Count() int {
	var count int

	for _, item := range c.items {
		count += item.Quantity
	}

	return count
}

func (c *Cart) IsEmpty() bool {
	return len(c.items) == 0
}

func (c *Cart) ClearCart() {
	c.items = make(map[string]*CartItem)
	c.order = nil
}

func (c *Cart) FormattedTotal() string {
	return fmt.Sprintf("$%.2f", c.Total())
}
