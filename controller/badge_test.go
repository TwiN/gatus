package controller

import (
	"strconv"
	"testing"
)

func TestGetBadgeColorFromUptime(t *testing.T) {
	scenarios := []struct {
		Uptime        float64
		ExpectedColor string
	}{
		{
			Uptime:        1,
			ExpectedColor: badgeColorHexAwesome,
		},
		{
			Uptime:        0.99,
			ExpectedColor: badgeColorHexAwesome,
		},
		{
			Uptime:        0.97,
			ExpectedColor: badgeColorHexGreat,
		},
		{
			Uptime:        0.95,
			ExpectedColor: badgeColorHexGreat,
		},
		{
			Uptime:        0.93,
			ExpectedColor: badgeColorHexGood,
		},
		{
			Uptime:        0.9,
			ExpectedColor: badgeColorHexGood,
		},
		{
			Uptime:        0.85,
			ExpectedColor: badgeColorHexPassable,
		},
		{
			Uptime:        0.7,
			ExpectedColor: badgeColorHexBad,
		},
		{
			Uptime:        0.65,
			ExpectedColor: badgeColorHexBad,
		},
		{
			Uptime:        0.6,
			ExpectedColor: badgeColorHexVeryBad,
		},
	}
	for _, scenario := range scenarios {
		t.Run("uptime-"+strconv.Itoa(int(scenario.Uptime*100)), func(t *testing.T) {
			if getBadgeColorFromUptime(scenario.Uptime) != scenario.ExpectedColor {
				t.Errorf("expected %s from %f, got %v", scenario.ExpectedColor, scenario.Uptime, getBadgeColorFromUptime(scenario.Uptime))
			}
		})
	}
}

func TestGetBadgeColorFromResponseTime(t *testing.T) {
	scenarios := []struct {
		ResponseTime  int
		ExpectedColor string
	}{
		{
			ResponseTime:  10,
			ExpectedColor: badgeColorHexAwesome,
		},
		{
			ResponseTime:  50,
			ExpectedColor: badgeColorHexAwesome,
		},
		{
			ResponseTime:  75,
			ExpectedColor: badgeColorHexGreat,
		},
		{
			ResponseTime:  150,
			ExpectedColor: badgeColorHexGreat,
		},
		{
			ResponseTime:  201,
			ExpectedColor: badgeColorHexGood,
		},
		{
			ResponseTime:  300,
			ExpectedColor: badgeColorHexGood,
		},
		{
			ResponseTime:  301,
			ExpectedColor: badgeColorHexPassable,
		},
		{
			ResponseTime:  450,
			ExpectedColor: badgeColorHexPassable,
		},
		{
			ResponseTime:  700,
			ExpectedColor: badgeColorHexBad,
		},
		{
			ResponseTime:  1500,
			ExpectedColor: badgeColorHexVeryBad,
		},
	}
	for _, scenario := range scenarios {
		t.Run("response-time-"+strconv.Itoa(scenario.ResponseTime), func(t *testing.T) {
			if getBadgeColorFromResponseTime(scenario.ResponseTime) != scenario.ExpectedColor {
				t.Errorf("expected %s from %d, got %v", scenario.ExpectedColor, scenario.ResponseTime, getBadgeColorFromResponseTime(scenario.ResponseTime))
			}
		})
	}
}
