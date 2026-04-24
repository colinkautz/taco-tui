package main

import (
	"fmt"
	"strings"

	"charm.land/lipgloss/v2"
)

func renderItemRow(
	entry *CartItem,
	nameW, qtyW, priceW int,
	nameStyle, qtyStyle, priceStyle lipgloss.Style,
) string {
	nameLines := wrapText(entry.Item.Name, nameW)
	qty := fmt.Sprintf("x%d", entry.Quantity)
	price := fmt.Sprintf("$%.2f", entry.Item.Price*float64(entry.Quantity))

	qtyCell := lipgloss.NewStyle().Width(qtyW).Align(lipgloss.Right).Render(qtyStyle.Render(qty))

	var firstLine string
	if priceW > 0 {
		priceCell := lipgloss.NewStyle().Width(priceW).Align(lipgloss.Right).Render(priceStyle.Render(price))
		firstLine = fmt.Sprintf("%-*s %s %s", nameW, nameLines[0], qtyCell, priceCell)
	} else {
		firstLine = fmt.Sprintf("%-*s %s", nameW, nameLines[0], qtyCell)
	}

	var lines []string
	lines = append(lines, nameStyle.Render(firstLine))
	for _, nameLine := range nameLines[1:] {
		lines = append(lines, nameStyle.Render(nameLine))
	}

	return strings.Join(lines, "\n")
}
