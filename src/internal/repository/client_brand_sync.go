package repository

import (
	"time"

	persistencemodels "b2b-diagnostic-aggregator/apis/internal/persistence/models"

	"gorm.io/gorm"
)

// syncClientBrandMappings reconciles MediAdmin.tbl_ClientBrandMapping with desired brand names.
// desired must be non-empty and already normalized (trim, dedupe). Rows not in desired are soft-deactivated (IsActive=false).
// Names in desired that already exist (any row) are left unchanged if active, or reactivated with audit fields if inactive.
// Names in desired with no row are inserted.
func syncClientBrandMappings(tx *gorm.DB, clientID int64, desired []string, clientIsActive bool, lastUpdatedBy int64, lastUpdatedOn time.Time) error {
	var existing []persistencemodels.ClientBrandMapping
	if err := tx.Where("ClientID = ?", clientID).Find(&existing).Error; err != nil {
		return err
	}

	desiredSet := make(map[string]struct{}, len(desired))
	for _, n := range desired {
		desiredSet[n] = struct{}{}
	}

	for i := range existing {
		row := &existing[i]
		if _, keep := desiredSet[row.BrandName]; keep {
			continue
		}
		if !row.IsActive {
			continue
		}
		if err := tx.Model(&persistencemodels.ClientBrandMapping{}).
			Where("UID = ?", row.UID).
			Updates(map[string]interface{}{
				"IsActive":        false,
				"LastUpdatedBy":   lastUpdatedBy,
				"LastUpdatedOn":   lastUpdatedOn,
			}).Error; err != nil {
			return err
		}
		row.IsActive = false
	}

	byName := make(map[string][]*persistencemodels.ClientBrandMapping)
	for i := range existing {
		row := &existing[i]
		byName[row.BrandName] = append(byName[row.BrandName], row)
	}

	for _, name := range desired {
		rows := byName[name]
		if len(rows) == 0 {
			createdRow := persistencemodels.ClientBrandMapping{
				ClientID:        clientID,
				BrandName:       name,
				IsActive:        clientIsActive,
				CreatedBy:       lastUpdatedBy,
				CreatedOn:       lastUpdatedOn,
				LastUpdatedBy:   lastUpdatedBy,
				LastUpdatedOn:   lastUpdatedOn,
			}
			if err := tx.Create(&createdRow).Error; err != nil {
				return err
			}
			byName[name] = append(byName[name], &createdRow)
			continue
		}
		for _, row := range rows {
			if row.IsActive {
				continue
			}
			if err := tx.Model(&persistencemodels.ClientBrandMapping{}).
				Where("UID = ?", row.UID).
				Updates(map[string]interface{}{
					"IsActive":        true,
					"LastUpdatedBy":   lastUpdatedBy,
					"LastUpdatedOn":   lastUpdatedOn,
				}).Error; err != nil {
				return err
			}
			row.IsActive = true
		}
	}
	return nil
}
