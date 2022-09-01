### Fetch secrets from AWS Secrets Manager

#### Usage

This small binary fetches JSON formatted secrets from AWS Secrets Manager.  

Provide the `SECRET_PATH` env variable to configure the fetching location.  
It should usually follow the pattern `$service_name/$env/secrets`  

You then run it like this:  
```sh
./fetch-secrets mybinary subcommand --my-arg example
```


### Building

Run the following commands:  
```sh
go get
go build
````

The resulting binary will be in this directory called `fetch-secrets`.
