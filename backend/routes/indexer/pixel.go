package indexer

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/AFK-AlignedFamKernel/afk_monorepo/backend/core"
	routeutils "github.com/AFK-AlignedFamKernel/afk_monorepo/backend/routes/utils"
)

func processPixelPlacedEvent(event IndexerEvent) {
	address := event.Event.Keys[1][2:] // Remove 0x prefix
	posHex := event.Event.Keys[2]
	dayIdxHex := event.Event.Keys[3]
	colorHex := event.Event.Data[0]

	// Convert hex to int
	position, err := strconv.ParseInt(posHex, 0, 64)
	if err != nil {
		PrintIndexerError("processPixelPlacedEvent", "Error converting position hex to int", address, posHex, dayIdxHex, colorHex)
		return
	}

	//validate position
	maxPosition := int64(core.AFKBackend.CanvasConfig.Canvas.Width) * int64(core.AFKBackend.CanvasConfig.Canvas.Height)

	// Perform comparison with maxPosition
	if position < 0 || position >= maxPosition {
		PrintIndexerError("processPixelPlacedEvent", "Position value exceeds canvas dimensions", address, posHex, dayIdxHex, colorHex)
		return
	}

	dayIdx, err := strconv.ParseInt(dayIdxHex, 0, 64)
	if err != nil {
		PrintIndexerError("processPixelPlacedEvent", "Error converting day index hex to int", address, posHex, dayIdxHex, colorHex)
		return
	}
	color, err := strconv.ParseInt(colorHex, 0, 64)
	if err != nil {
		PrintIndexerError("processPixelPlacedEvent", "Error converting color hex to int", address, posHex, dayIdxHex, colorHex)
		return
	}

	// Set pixel in redis
	bitfieldType := "u" + strconv.Itoa(int(core.AFKBackend.CanvasConfig.ColorsBitWidth))
	pos := uint(position) * core.AFKBackend.CanvasConfig.ColorsBitWidth

	ctx := context.Background()
	err = core.AFKBackend.Databases.Redis.BitField(ctx, "canvas", "SET", bitfieldType, pos, color).Err()
	if err != nil {
		PrintIndexerError("processPixelPlacedEvent", "Error setting pixel in redis", address, posHex, dayIdxHex, colorHex)
		return
	}

	fmt.Printf(address, position, dayIdx, color, "print")
	// Set pixel in postgres
	_, err = core.AFKBackend.Databases.Postgres.Exec(context.Background(), "INSERT INTO Pixels (address, position, day, color) VALUES ($1, $2, $3, $4)", address, position, dayIdx, color)
	if err != nil {
		// TODO: Reverse redis operation?
		PrintIndexerError("processPixelPlacedEvent", "Error inserting pixel into postgres", address, posHex, dayIdxHex, colorHex)
		return
	}

	// Send message to all connected clients
	var message = map[string]interface{}{
		"position":    position,
		"color":       color,
		"messageType": "colorPixel",
	}
	routeutils.SendWebSocketMessage(message)
}

func revertPixelPlacedEvent(event IndexerEvent) {
	address := event.Event.Keys[1][2:] // Remove 0x prefix
	posHex := event.Event.Keys[2]

	// Convert hex to int
	position, err := strconv.ParseInt(posHex, 0, 64)
	if err != nil {
		PrintIndexerError("revertPixelPlacedEvent", "Error converting position hex to int", address, posHex)
		return
	}

	// Delete pixel from postgres ( last one )
	_, err = core.AFKBackend.Databases.Postgres.Exec(context.Background(), "DELETE FROM Pixels WHERE address = $1 AND position = $2 ORDER BY time limit 1", address, position)
	if err != nil {
		PrintIndexerError("revertPixelPlacedEvent", "Error deleting pixel from postgres", address, posHex)
		return
	}

	// Retrieve the old color
	oldColor, err := core.PostgresQueryOne[int]("SELECT color FROM Pixels WHERE address = $1 AND position = $2 ORDER BY time DESC LIMIT 1", address, position)
	if err != nil {
		PrintIndexerError("revertPixelPlacedEvent", "Error retrieving old color from postgres", address, posHex)
		return
	}
	// Reset pixel in redis
	bitfieldType := "u" + strconv.Itoa(int(core.AFKBackend.CanvasConfig.ColorsBitWidth))
	pos := uint(position) * core.AFKBackend.CanvasConfig.ColorsBitWidth

	ctx := context.Background()
	err = core.AFKBackend.Databases.Redis.BitField(ctx, "canvas", "SET", bitfieldType, pos, oldColor).Err()
	if err != nil {
		PrintIndexerError("revertPixelPlacedEvent", "Error resetting pixel in redis", address, posHex)
		return
	}

	// Send message to all connected clients
	var message = map[string]interface{}{
		"position":    position,
		"color":       oldColor,
		"messageType": "colorPixel",
	}
	routeutils.SendWebSocketMessage(message)
}

func processBasicPixelPlacedEvent(event IndexerEvent) {
	address := event.Event.Keys[1][2:] // Remove 0x prefix
	timestampHex := event.Event.Data[0]
	timestamp, err := strconv.ParseInt(timestampHex, 0, 64)
	if err != nil {
		PrintIndexerError("processBasicPixelPlacedEvent", "Error converting timestamp hex to int", address, timestampHex)
		return
	}

	_, err = core.AFKBackend.Databases.Postgres.Exec(context.Background(), "INSERT INTO LastPlacedTime (address, time) VALUES ($1, TO_TIMESTAMP($2)) ON CONFLICT (address) DO UPDATE SET time = TO_TIMESTAMP($2)", address, timestamp)
	if err != nil {
		PrintIndexerError("processBasicPixelPlacedEvent", "Error inserting last placed time into postgres", address, timestampHex)
		return
	}
}

func revertBasicPixelPlacedEvent(event IndexerEvent) {
	address := event.Event.Keys[1][2:] // Remove 0x prefix

	// Reset last placed time to time of last pixel placed
	_, err := core.AFKBackend.Databases.Postgres.Exec(context.Background(), "UPDATE LastPlacedTime SET time = (SELECT time FROM Pixels WHERE address = $1 ORDER BY time DESC LIMIT 1) WHERE address = $1", address)
	if err != nil {
		PrintIndexerError("revertBasicPixelPlacedEvent", "Error resetting last placed time in postgres", address)
		return
	}

	// TODO: check ordering of this and revertPixelPlacedEvent
}

func processFactionPixelsPlacedEvent(event IndexerEvent) {
	// TODO: Faction id
	userAddress := event.Event.Keys[1][2:] // Remove 0x prefix
	timestampHex := event.Event.Data[0]
	memberPixelsHex := event.Event.Data[1]

	timestamp, err := strconv.ParseInt(timestampHex, 0, 64)
	if err != nil {
		PrintIndexerError("processMemberPixelsPlacedEvent", "Error converting timestamp hex to int", userAddress, timestampHex, memberPixelsHex)
		return
	}

	memberPixels, err := strconv.ParseInt(memberPixelsHex, 0, 64)
	if err != nil {
		PrintIndexerError("processMemberPixelsPlacedEvent", "Error converting member pixels hex to int", userAddress, timestampHex, memberPixelsHex)
		return
	}

	_, err = core.AFKBackend.Databases.Postgres.Exec(context.Background(), "UPDATE FactionMembersInfo SET last_placed_time = TO_TIMESTAMP($1), member_pixels = $2 WHERE user_address = $3", timestamp, memberPixels, userAddress)
	if err != nil {
		PrintIndexerError("processMemberPixelsPlacedEvent", "Error updating faction member info in postgres", userAddress, timestampHex, memberPixelsHex)
		return
	}
}

func revertFactionPixelsPlacedEvent(event IndexerEvent) {
	// TODO
}

func processChainFactionPixelsPlacedEvent(event IndexerEvent) {
	// TODO: Faction id
	userAddress := event.Event.Keys[1][2:] // Remove 0x prefix
	timestampHex := event.Event.Data[0]
	memberPixelsHex := event.Event.Data[1]

	timestamp, err := strconv.ParseInt(timestampHex, 0, 64)
	if err != nil {
		PrintIndexerError("processChainFactionMemberPixelsPlacedEvent", "Error converting timestamp hex to int", userAddress, timestampHex, memberPixelsHex)
		return
	}

	memberPixels, err := strconv.ParseInt(memberPixelsHex, 0, 64)
	if err != nil {
		PrintIndexerError("processChainFactionMemberPixelsPlacedEvent", "Error converting member pixels hex to int", userAddress, timestampHex, memberPixelsHex)
		return
	}

	_, err = core.AFKBackend.Databases.Postgres.Exec(context.Background(), "UPDATE ChainFactionMembersInfo SET last_placed_time = TO_TIMESTAMP($1), member_pixels = $2 WHERE user_address = $3", timestamp, memberPixels, userAddress)
	if err != nil {
		PrintIndexerError("processChainFactionMemberPixelsPlacedEvent", "Error updating chain faction member info in postgres", userAddress, timestampHex, memberPixelsHex)
		return
	}
}

func revertChainFactionPixelsPlacedEvent(event IndexerEvent) {
	// TODO
}

func processExtraPixelsPlacedEvent(event IndexerEvent) {
	address := event.Event.Keys[1][2:] // Remove 0x prefix
	extraPixelsHex := event.Event.Data[0]

	extraPixels, err := strconv.ParseInt(extraPixelsHex, 0, 64)
	if err != nil {
		PrintIndexerError("processExtraPixelsPlacedEvent", "Error converting extra pixels hex to int", address, extraPixelsHex)
		return
	}

	_, err = core.AFKBackend.Databases.Postgres.Exec(context.Background(), "UPDATE ExtraPixels SET available = available - $1, used = used + $1 WHERE address = $2", extraPixels, address)
	if err != nil {
		PrintIndexerError("processExtraPixelsPlacedEvent", "Error updating extra pixels in postgres", address, extraPixelsHex)
		return
	}
}

func revertExtraPixelsPlacedEvent(event IndexerEvent) {
	address := event.Event.Keys[1][2:] // Remove 0x prefix
	extraPixelsHex := event.Event.Data[0]

	extraPixels, err := strconv.ParseInt(extraPixelsHex, 0, 64)
	if err != nil {
		PrintIndexerError("revertExtraPixelsPlacedEvent", "Error converting extra pixels hex to int", address, extraPixelsHex)
		return
	}

	_, err = core.AFKBackend.Databases.Postgres.Exec(context.Background(), "UPDATE ExtraPixels SET available = available + $1, used = used - $1 WHERE address = $2", extraPixels, address)
	if err != nil {
		PrintIndexerError("revertExtraPixelsPlacedEvent", "Error updating extra pixels in postgres", address, extraPixelsHex)
		return
	}
}

func processBasicPixelPlacedEventWithMetadata(event IndexerEvent) {
	address := event.Event.Keys[1][2:] // Remove 0x prefix
	timestampHex := event.Event.Data[0]
	timestamp, err := strconv.ParseInt(timestampHex, 0, 64)
	if err != nil {
		PrintIndexerError("processBasicPixelPlacedEventWithMetadata", "Error converting timestamp hex to int", address, timestampHex)
		return
	}

	// Extract position and color from the event (position is Keys[2], color is in Data[1])
	positionHex := event.Event.Keys[2]
	position, err := strconv.Atoi(positionHex)
	if err != nil {
		PrintIndexerError("processBasicPixelPlacedEventWithMetadata", "Error converting position hex to int", address, positionHex)
		return
	}

	colorHex := event.Event.Data[1]
	color, err := strconv.Atoi(colorHex)
	if err != nil {
		PrintIndexerError("processBasicPixelPlacedEventWithMetadata", "Error converting color hex to int", address, colorHex)
		return
	}

	// Extract metadata from the last index in Data (metadata is in Data[n])
	metadata := event.Event.Data[len(event.Event.Data)-1]

	// Unmarshal metadata (if it exists)
	var metadataMap map[string]interface{}
	if len(metadata) > 0 {
		err = json.Unmarshal([]byte(metadata), &metadataMap)
		if err != nil {
			PrintIndexerError("processBasicPixelPlacedEventWithMetadata", "Error parsing metadata", address, string(metadata))
			return
		}
	}

	// Prepare SQL statement for inserting pixel info and metadata together
	metadataJson, err := json.Marshal(metadataMap)
	if err != nil {
		PrintIndexerError("processBasicPixelPlacedEventWithMetadata", "Error serializing metadata", address, string(metadata))
		return
	}

	// Use a single query to insert the pixel information and metadata into the database
	_, err = core.AFKBackend.Databases.Postgres.Exec(context.Background(),
		`INSERT INTO Pixels (address, position, color, time)
		 VALUES ($1, $2, $3, TO_TIMESTAMP($4))
		 ON CONFLICT (address, position)
		 DO UPDATE SET color = $3, time = TO_TIMESTAMP($4),
		 metadata = COALESCE(metadata, $5)`,
		address, position, color, timestamp, metadataJson)
	if err != nil {
		PrintIndexerError("processBasicPixelPlacedEventWithMetadata", "Error inserting/updating pixel and metadata", address, string(metadataJson))
		return
	}

	// Insert or update the last placed time in the LastPlacedTime table
	_, err = core.AFKBackend.Databases.Postgres.Exec(context.Background(),
		"INSERT INTO LastPlacedTime (address, time) VALUES ($1, TO_TIMESTAMP($2)) ON CONFLICT (address) DO UPDATE SET time = TO_TIMESTAMP($2)",
		address, timestamp)
	if err != nil {
		PrintIndexerError("processBasicPixelPlacedEventWithMetadata", "Error inserting last placed time into postgres", address, timestampHex)
		return
	}
}

func revertBasicPixelPlacedEventWithMetadata(event IndexerEvent) {
	address := event.Event.Keys[1][2:] // Remove 0x prefix
	posHex := event.Event.Keys[2]

	// Convert hex to int for position
	position, err := strconv.ParseInt(posHex, 0, 64)
	if err != nil {
		PrintIndexerError("revertPixelPlacedEvent", "Error converting position hex to int", address, posHex)
		return
	}

	// We can also retrieve the metadata from the event if needed
	metadata := event.Event.Data[len(event.Event.Data)-1]
	var metadataMap map[string]interface{}
	if len(metadata) > 0 {
		err = json.Unmarshal([]byte(metadata), &metadataMap) // Unmarshal from metadata (which is a string) to map
		if err != nil {
			PrintIndexerError("revertPixelPlacedEvent", "Error parsing metadata", address, string(metadata))
			return
		}
	}

	// Delete the pixel entry (including metadata) from the PostgreSQL database
	_, err = core.AFKBackend.Databases.Postgres.Exec(context.Background(), `
		DELETE FROM Pixels
		WHERE address = $1 AND position = $2
		ORDER BY time LIMIT 1`, address, position)
	if err != nil {
		PrintIndexerError("revertPixelPlacedEvent", "Error deleting pixel from postgres", address, posHex)
		return
	}

	// Optionally, you can also delete the metadata from the database,
	// but usually deleting the pixel entry will automatically take care of it since metadata is part of the same row.

	// Delete the pixel's associated last placed time entry from the LastPlacedTime table
	_, err = core.AFKBackend.Databases.Postgres.Exec(context.Background(),
		"DELETE FROM LastPlacedTime WHERE address = $1", address)
	if err != nil {
		PrintIndexerError("revertPixelPlacedEvent", "Error deleting last placed time from postgres", address, posHex)
		return
	}

	// Optionally log the event if needed
	fmt.Printf("Pixel at position %d for address %s has been reverted.\n", position, address)
}
