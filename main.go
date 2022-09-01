package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/exec"
	"syscall"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/secretsmanager"
)

func main() {

	var secretName string = os.Getenv("SECRET_PATH")

	//Create a Secrets Manager client
	sess, err := session.NewSession()
	if err != nil {
		// Handle session creation error
		fmt.Println(err.Error())
		panic(err)
	}
	svc := secretsmanager.New(sess)
	input := &secretsmanager.GetSecretValueInput{
		SecretId: aws.String(secretName),
	}

	result, err := svc.GetSecretValue(input)
	if err != nil {
		if err, ok := err.(awserr.Error); ok {
			fmt.Println(err.Error())
			panic(err)
		}
	}

	var secretString string
	secretString = *result.SecretString

	var anyJson interface{}

	json.Unmarshal([]byte(secretString), &anyJson)

	path, err := exec.LookPath(os.Args[1])

	var updatedEnv []string = os.Environ()

	for key, el := range anyJson.(map[string]interface{}) {
		updatedEnv = append(updatedEnv, fmt.Sprintf("%s=%s", key, el))
	}

	if err := syscall.Exec(path, os.Args[1:], updatedEnv); err != nil {
		log.Fatal(err)
	}

}
