package main

import "log"

func hnHandler(message Message) {

	claims, err := decodeJWT(message.Token)

	if err != nil {
		log.Println(err)
		return
	}

	log.Printf("Received message from '%s' of type '%s': %s\n", claims["name"], message.Type, message.Text)
}
