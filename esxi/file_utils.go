package esxi

import (
	"io"
	"log"
	"os"
)

func writeStringToFile(fileName string, data string) error {
	file, err := os.Create(fileName)
	if err != nil {
		log.Println("[writeStringToFile] Failed to create file " + fileName + " | " + err.Error())
		return err
	}
	defer file.Close()

	_, err = io.WriteString(file, data)
	if err != nil {
		log.Println("[writeStringToFile] Failed to write data to file " + data + " | " + err.Error())
		return err
	}
	return file.Sync()
}

func deleteFile(fileName string) error {
	err := os.Remove(fileName)

	if err != nil {
		log.Println("[deleteFile] Failed to delete local file " + fileName + " | " + err.Error())
	}

	return err
}
