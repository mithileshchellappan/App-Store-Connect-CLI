package validate

import (
	"context"
	"fmt"
	"strings"

	"github.com/rudrankriyam/App-Store-Connect-CLI/internal/asc"
	"github.com/rudrankriyam/App-Store-Connect-CLI/internal/cli/shared"
)

func fetchAvailableTerritories(ctx context.Context, client *asc.Client, appID string) (string, int, error) {
	availabilityID := ""
	availableTerritories := 0

	availabilityResp, err := client.GetAppAvailabilityV2(ctx, appID)
	if err != nil {
		if shared.IsAppAvailabilityMissing(err) {
			return "", 0, nil
		}
		return "", 0, fmt.Errorf("failed to fetch app availability: %w", err)
	}

	availabilityID = strings.TrimSpace(availabilityResp.Data.ID)
	if availabilityID == "" {
		return "", 0, nil
	}

	nextURL := ""
	for {
		var territoryResp *asc.TerritoryAvailabilitiesResponse
		if strings.TrimSpace(nextURL) != "" {
			territoryResp, err = client.GetTerritoryAvailabilities(ctx, availabilityID, asc.WithTerritoryAvailabilitiesNextURL(nextURL))
		} else {
			territoryResp, err = client.GetTerritoryAvailabilities(ctx, availabilityID, asc.WithTerritoryAvailabilitiesLimit(200))
		}
		if err != nil {
			return availabilityID, availableTerritories, fmt.Errorf("failed to fetch territory availabilities: %w", err)
		}

		for _, territoryAvailability := range territoryResp.Data {
			if territoryAvailability.Attributes.Available {
				availableTerritories++
			}
		}

		nextURL = strings.TrimSpace(territoryResp.Links.Next)
		if nextURL == "" {
			break
		}
	}

	return availabilityID, availableTerritories, nil
}
