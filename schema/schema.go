package main

import (
        "encoding/json"
        "fmt"
        "github.com/invopop/jsonschema"
        "inspector/config"
        "os"
)

func main() {
        schemaJSON, err := json.MarshalIndent(jsonschema.Reflect(&config.Config{}), "", "  ")
        if err != nil {
                fmt.Println("Error marshaling JSON schema:", err)
                return
        }
        if file, err := os.Create("schema.json"); err != nil {
                fmt.Println("Error creating file:", err)
        } else {
                defer file.Close()
                if _, err := file.Write(schemaJSON); err != nil {
                        fmt.Println("Error writing to file:", err)
                } else {
                        fmt.Println("JSON schema successfully saved to schema.json")
                }
        }
}
