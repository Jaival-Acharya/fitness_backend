package models

import "time"

// BMISession tracks BMI calculations over time
type BMISession struct {
	ID        int       `json:"id" gorm:"primaryKey"`
	UserID    int       `json:"user_id" gorm:"index"`
	Height    float64   `json:"height"` // in cm
	Weight    float64   `json:"weight"` // in kg
	BMIValue  float64   `json:"bmi_value"`
	Category  string    `json:"category"` // 'underweight', 'normal', 'overweight', 'obese'
	CreatedAt time.Time `json:"created_at" gorm:"index"`
}

func (BMISession) TableName() string {
	return "bmi_sessions"
}
