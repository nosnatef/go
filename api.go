package main

import (
	"errors"
	"math"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type Receipt struct {
	Retailer			string 		`json:"retailer" binding:"required"`
	PurchaseDate	string 		`json:"purchaseDate" binding:"required"`
	PurchaseTime	string 		`json:"purchaseTime" binding:"required"`
	Items					[]Item 		`json:"items" binding:"required,min=1"`
	Total					string 		`json:"total" binding:"required"`
}

type Item struct {
	ShortDescription	string 	`json:"shortDescription" binding:"required"`
	Price 						string `json:"price" binding:"required"`
}

var (
	receipts = make(map[string]Receipt)
	mapMutex	sync.Mutex
)

func main() {
	route := gin.Default()

	route.POST("/receipts/process", processReceipt)
	route.GET("/receipts/:id/points", getReceiptPoints)

	route.Run(":8080")
}

func getReceiptPoints(c *gin.Context) {
	receiptId := c.Param("id")

	mapMutex.Lock()
	receipt, exists := receipts[receiptId]
	mapMutex.Unlock()

	if !exists {
		c.JSON(http.StatusNotFound, gin.H{"description": "No receipt found for that ID."})
		return
	}

	totalPoints := calculatePoints(receipt)

	c.JSON(http.StatusOK, gin.H{"points": totalPoints})
}

// Calculating with custom calculator, allowing the rules to be updated more easily
func calculatePoints(receipt Receipt) int {
	totalPoints := 0

	totalPoints += calculatePointsForRetailerName(receipt.Retailer)

	totalPoints += calcuatePointsForTotal(receipt.Total)

	totalPoints += calculatePointsForItems(receipt.Items)

	totalPoints += calculatePointsForPurchaseDate(receipt.PurchaseDate)

	totalPoints += calculatePointsForPurchaseTime(receipt.PurchaseTime)

	return totalPoints
}

func calculatePointsForRetailerName(s string) int {
	points := 0

	// Rule 1
	for _, c := range s {
		if (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') {
			points += 1
		}
	}

	return points
}

func calcuatePointsForTotal(t string) int {
	points := 0

	total, _ := strconv.ParseFloat(t, 64)

	// Rule 2
	if almostEqual(total, float64(int(total))) {
		points += 50
	}

	// Rule 3
	if almostEqual(math.Mod(total, 0.25), 0) {
		points += 25
	}

	return points
}

func calculatePointsForItems(items []Item) int {
	points := 0

	// Rule 4
	points += (len(items) / 2) * 5

	// Rule 5
	for _, item := range items {
		description := strings.TrimSpace(item.ShortDescription)
		if len(description) % 3 == 0 {
			price, _ := strconv.ParseFloat(item.Price, 64)
			points += int(math.Ceil(price * 0.2))
		}
	}

	return points
}

func calculatePointsForPurchaseDate(d string) int {
	points := 0

	date, _ := time.Parse("2006-01-02", d)

	// Rule 7
	if date.Day() % 2 == 1 {
		points += 6
	}

	return points
}

func calculatePointsForPurchaseTime(t string) int {
	points := 0

	time, _ := time.Parse("15:04", t)

	// Rule 8
	if time.Hour() >= 14 && time.Hour() <= 16 {
		points += 10
	}

	return points
}

func processReceipt(c *gin.Context) {
	var receipt Receipt

	if err := c.ShouldBindJSON(&receipt); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"description": "The receipt is invalid."})
		return
	}

	if err := validateReceipt(receipt); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"description": "The receipt is invalid."})
		return
	}

	receiptId := uuid.New().String()

	mapMutex.Lock()
	receipts[receiptId] = receipt
	mapMutex.Unlock()

	c.JSON(http.StatusOK, gin.H{"id": receiptId})
}

// Check date format, total and price format, and if price adds up to total
func validateReceipt(receipt Receipt) error {
	if _, err := time.Parse("2006-01-02", receipt.PurchaseDate); err != nil {
		return err
	}

	if _, err := time.Parse("15:04", receipt.PurchaseTime); err != nil {
		return err
	}

	total, err := strconv.ParseFloat(receipt.Total, 64)
	if err != nil {
		return err
	}

	var sum float64
	for _, item := range receipt.Items {
		price, err := strconv.ParseFloat(item.Price, 64)
		if err != nil {
			return err
		}
		sum += price
	}

	if !almostEqual(total, sum) {
		return errors.New("")
	}

	return nil
}

// Dealing with float64 comparison
func almostEqual(a, b float64) bool {
	return math.Abs(a - b) <= 1e-9
}