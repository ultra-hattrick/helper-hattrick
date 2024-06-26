package utils

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/dghubble/oauth1"
	"github.com/gin-gonic/gin"
	"gitlab.com/uchile1/helper/helperCommon"
	"gitlab.com/uchile1/helper/helperLog"
)

func IsValidWeeks(weeks int) bool {
	validWeeks := []int{3, 5, 10, 15, 20}
	for _, w := range validWeeks {
		if w == weeks {
			return true
		}
	}
	return false
}

func IsValidStadium(stadium int) bool {
	valid := []int{1, 2}
	for _, w := range valid {
		if w == stadium {
			return true
		}
	}
	return false
}

// helper para obtener y validar un booleano de la consulta
func GetQueryBool(c *gin.Context, key string, defaultValue bool) (bool, error) {
	v, exists := c.GetQuery(key)
	if !exists {
		return defaultValue, nil
	}
	return strconv.ParseBool(v)
}

// helper para obtener y validar una matriz de cadenas de la consulta
func GetQueryStringArray(c *gin.Context, key string, defaultValue []string) ([]string, error) {
	v, exists := c.GetQuery(key)
	if !exists {
		return defaultValue, nil
	}
	arr := strings.Split(v, ",")
	for _, item := range arr {
		if _, err := strconv.Atoi(item); err != nil {
			return nil, fmt.Errorf("invalid value for %s parameter: %v", key, err)
		}
	}
	return arr, nil
}

// helper para obtener y validar un entero de la consulta
func GetQueryInt(c *gin.Context, key string, defaultValue int, validator func(int) bool) (int, error) {
	v, exists := c.GetQuery(key)
	if !exists {
		return defaultValue, nil
	}
	intValue, err := strconv.Atoi(v)
	if err != nil {
		return 0, fmt.Errorf("invalid %s parameter: %v", key, err)
	}
	if validator != nil && !validator(intValue) {
		return 0, fmt.Errorf("invalid value for %s parameter: %d", key, intValue)
	}
	return intValue, nil
}

func GetResultsFromHattrick(pathHattrick string, v any) error {
	config := oauth1.NewConfig(os.Getenv("CONSUMER_KEY"), os.Getenv("CONSUMER_SECRET"))
	httpClient := config.Client(oauth1.NoContext, oauth1.NewToken(os.Getenv("OAUTH1_TOKEN"), os.Getenv("OAUTH1_TOKEN_SECRET")))
	path := fmt.Sprintf("%s%s", os.Getenv("BASE_RESOURCE_URL"), pathHattrick)
	resp, err := httpClient.Get(path)
	helperLog.Logger.Warn().Str(
		"function", helperCommon.GetFrame(2).Function,
	).Msgf("Se ocupa API Hattrick url: %s", path)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// Imprimir el status code
	helperLog.Logger.Debug().Msgf("HTTP Status Code: %d para la url: %s", resp.StatusCode, path)
	if resp.StatusCode != 200 {
		return fmt.Errorf("HTTP Status Code: %d para la url: %s", resp.StatusCode, path)
	}

	// Leer el contenido del body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	helperLog.Logger.Debug().Msgf("Response Body: %s", string(body))

	// Reiniciar el cuerpo de la respuesta para que pueda ser decodificado
	resp.Body = io.NopCloser(bytes.NewBuffer(body))

	err = xml.NewDecoder(resp.Body).Decode(v)
	if err != nil {
		return err
	}
	// helperLog.Logger.Debug().Msgf("--->Arena: %v", hattrickData.Match.DetailsMatch.Arena)
	return nil
}

// GetLastDayAndPlusDays returns two dates:
// 1. Last Friday with the time set to START_TIME_TRAINING_UTC
// 2. The date of the first date plus 3 days (time.Hour * 72) for example
func GetLastDayAndPlusDuration(weekDayUTC time.Weekday, plusDuration time.Duration, timeIntUTC string) (time.Time, time.Time, error) {
	// Get the current time
	now := time.Now().In(time.UTC)

	// Find the last Friday in UTC
	helperLog.Logger.Debug().Msgf("now.Weekday(): %d , now: %v", now.Weekday(), now)
	daysSinceFriday := (int(now.Weekday()) + 7 - int(weekDayUTC)) % 7
	helperLog.Logger.Debug().Msgf("daysSinceFriday: %v", daysSinceFriday)
	lastFridayUTC := now.AddDate(0, 0, -daysSinceFriday)

	// Parse START_TIME_TRAINING to get hour, minute, second
	startTimeParts := strings.Split(timeIntUTC, ":")
	if len(startTimeParts) != 3 {
		return time.Time{}, time.Time{}, fmt.Errorf("invalid START_TIME_TRAINING format")
	}
	hour, err := strconv.Atoi(startTimeParts[0])
	if err != nil {
		return time.Time{}, time.Time{}, fmt.Errorf("invalid hour in START_TIME_TRAINING")
	}
	minute, err := strconv.Atoi(startTimeParts[1])
	if err != nil {
		return time.Time{}, time.Time{}, fmt.Errorf("invalid minute in START_TIME_TRAINING")
	}
	second, err := strconv.Atoi(startTimeParts[2])
	if err != nil {
		return time.Time{}, time.Time{}, fmt.Errorf("invalid second in START_TIME_TRAINING")
	}

	// Set the last Friday date with START_TIME_TRAINING in UTC
	lastFridayUTC = time.Date(lastFridayUTC.Year(), lastFridayUTC.Month(), lastFridayUTC.Day(), hour, minute, second, 0, time.UTC)

	// Convert last Friday to local time
	location := time.Now().Location()
	lastFridayLocal := lastFridayUTC.In(location)

	plusThreeDaysLocal := lastFridayLocal.Add(plusDuration)

	return lastFridayLocal, plusThreeDaysLocal, nil
}
