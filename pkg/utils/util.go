package utils

import "github.com/google/uuid"

func GetUuid(uuidString string) uuid.UUID {
	log := GetSugaredLogger()
	result, err := uuid.Parse(uuidString)
	if err != nil {
		log.Warnf("invalid uuid: %s", uuidString)
		return uuid.New()
	}
	return result
}
