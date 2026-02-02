package repository

import (
	"b2b-diagnostic-aggregator/apis/internal/domain"
	persistencemodels "b2b-diagnostic-aggregator/apis/internal/persistence/models"
)

func mapLoginToDomain(p persistencemodels.Login) domain.Login {
	return domain.Login{
		RecordID:      p.RecordID,
		UserID:        p.UserID,
		Pwd:           p.Pwd,
		UserType:      p.UserType,
		CreatedOn:     p.CreatedOn,
		LastUpdatedOn: p.LastUpdatedOn,
	}
}
