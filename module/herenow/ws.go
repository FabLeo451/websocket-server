package herenow

import (
	"ekhoes-server/auth"
	"ekhoes-server/common"
	"ekhoes-server/utils"
	"encoding/json"
	"errors"
	"fmt"
)

type Query struct {
	Id         string     `json:"id"`
	Boundaries Boundaries `json:"boundaries"`
}

func WsHandler(user auth.User, in common.Message, out *common.Message) error {

	utils.Debug("Received message of type '%s': %s\n", in.Type, in.Payload)

	switch in.Type {
	case "query":

		var query Query

		err := json.Unmarshal(in.Payload, &query)

		if err != nil {
			return err
		}

		switch query.Id {
		case "getHotspotsByBoundaries":
			out.Type = "array"
			var hotspots, ephemerals []Hotspot

			//fmt.Printf("%+v\n", query.Boundaries)

			if err != nil {
				e := fmt.Sprintf("Error parsing boundaries string: %v\n", err)
				return errors.New(e)
			}

			hotspots = getHotspotsInBoundaries(user.Id, query.Boundaries)
			ephemerals = getEphemeralHotspots()
			hotspots = append(hotspots, ephemerals...)

			out.Payload, err = json.Marshal(hotspots)
			if err != nil {
				return err
			}

		default:
			e := fmt.Sprintf("Unespected query: %s\n", query.Id)
			return errors.New(e)
		}

		//fmt.Printf("Hotspots found: %d\n", len(hotspots))

	default:
		e := fmt.Sprintf("Unespected type: %s\n", in.Type)
		return errors.New(e)
	}

	return nil
}
