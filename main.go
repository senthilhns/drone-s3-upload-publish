package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	log "github.com/sirupsen/logrus"
	"github.com/urfave/cli"
)

var (
	pluginVersion = "1.0.0"
)

func main() {
	app := cli.NewApp()
	app.Name = "drone-s3-upload-publish"
	app.Usage = "Drone plugin to upload file/directories to AWS S3 Bucket and display the bucket url under 'Executions > Artifacts' tab"
	app.Action = run
	app.Version = pluginVersion
	app.Flags = []cli.Flag{
		cli.StringFlag{
			Name:   "aws-access-key",
			Usage:  "AWS Access Key ID",
			EnvVar: "PLUGIN_AWS_ACCESS_KEY_ID",
		},
		cli.StringFlag{
			Name:   "aws-secret-key",
			Usage:  "AWS Secret Access Key",
			EnvVar: "PLUGIN_AWS_SECRET_ACCESS_KEY",
		},
		cli.StringFlag{
			Name:   "aws-default-region",
			Usage:  "AWS Default Region",
			EnvVar: "PLUGIN_AWS_DEFAULT_REGION",
		},
		cli.StringFlag{
			Name:   "aws-bucket",
			Usage:  "AWS S3 Bucket",
			EnvVar: "PLUGIN_AWS_BUCKET",
		},
		cli.StringFlag{
			Name:   "source",
			Usage:  "Source",
			EnvVar: "PLUGIN_SOURCE",
		},
		cli.StringFlag{
			Name:   "target-path",
			Usage:  "target",
			EnvVar: "PLUGIN_TARGET",
		},
		cli.StringFlag{
			Name:   "artifact-file",
			Usage:  "Artifact file",
			EnvVar: "PLUGIN_ARTIFACT_FILE",
		},
		cli.StringFlag{
			Name:   "include",
			Usage:  "Include file patterns (comma-separated)",
			EnvVar: "PLUGIN_INCLUDE",
		},
		cli.StringFlag{
			Name:   "exclude",
			Usage:  "Exclude file patterns (comma-separated)",
			EnvVar: "PLUGIN_EXCLUDE",
		},
	}

	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
	}
}

func run(c *cli.Context) error {
	awsAccessKey := c.String("aws-access-key")
	awsSecretKey := c.String("aws-secret-key")
	awsDefaultRegion := c.String("aws-default-region")
	awsBucket := c.String("aws-bucket")
	source := c.String("source")
	target := c.String("target-path")
	newFolder := filepath.Base(source)
	artifactFilePath := c.String("artifact-file")
	includePatterns := c.String("include")
	excludePatterns := c.String("exclude")
	var urls string
	var dstS3Path string
	var argsList, includeArgsList, excludeArgsList []string

	if strings.ContainsAny(source, "*") {
		log.Fatal("Glob pattern not allowed!")
	}

	fileType, err := os.Stat(source)
	if err != nil {
		log.Fatal(err)
	}

	// AWS config commands to set ACCESS_KEY_ID and SECRET_ACCESS_KEY
	exec.Command("aws", "configure", "set", "aws_access_key_id", awsAccessKey).Run()
	exec.Command("aws", "configure", "set", "aws_secret_access_key", awsSecretKey).Run()

	var Uploadcmd *exec.Cmd

	if excludePatterns != "" {
		for _, pattern := range strings.Split(excludePatterns, ",") {
			excludeArgsList = append(excludeArgsList, "--exclude", strings.TrimSpace(pattern))
		}
	}

	if includePatterns != "" {
		for _, pattern := range strings.Split(includePatterns, ",") {
			includeArgsList = append(includeArgsList, "--include", strings.TrimSpace(pattern))
		}
	}

	if fileType.IsDir() {
		if target != "" {
			dstS3Path = "s3://" + awsBucket + "/" + target + "/" + newFolder
			urls = S3DirPathUrl + awsBucket + "?region=" + awsDefaultRegion + "&prefix=" + target + "/" + newFolder + "/&showversions=false"
		} else {
			dstS3Path = "s3://" + awsBucket + "/" + newFolder
			urls = S3DirPathUrl + awsBucket + "?region=" + awsDefaultRegion + "&prefix=" + newFolder + "/&showversions=false"
		}
		argsList = []string{"aws", "s3", "cp", source, dstS3Path, "--region", awsDefaultRegion, "--recursive"}
	} else {
		if target != "" {
			dstS3Path = "s3://" + awsBucket + "/" + target + "/" + newFolder
			urls = S3ObjPathUrl + awsBucket + "?region=" + awsDefaultRegion + "&prefix=" + target + "/" + newFolder
		} else {
			dstS3Path = "s3://" + awsBucket + "/"
			urls = S3ObjPathUrl + awsBucket + "?region=" + awsDefaultRegion + "&prefix=" + newFolder
		}
		argsList = []string{"s3", "cp", source, dstS3Path, "--region", awsDefaultRegion}
	}

	if len(includeArgsList) > 0 {
		argsList = append(argsList, ExcludeAllTypes...)
		argsList = append(argsList, includeArgsList...)
	}

	//fmt.Println("aws ", argsList)
	Uploadcmd = exec.Command("aws", argsList...)

	out, err := Uploadcmd.Output()
	if err != nil {
		fmt.Println(string(out))
		fmt.Println("Error uploading to S3 bucket", err.Error())
		return err
	}
	fmt.Printf("Output: %s\n", out)
	// End of S3 upload operation

	files := make([]File, 0)
	files = append(files, File{Name: artifactFilePath, URL: urls})

	return writeArtifactFile(files, artifactFilePath)
}

var ExcludeAllTypes = []string{"--exclude", "*"}

const S3DirPathUrl = "https://s3.console.aws.amazon.com/s3/buckets/"
const S3ObjPathUrl = "https://s3.console.aws.amazon.com/s3/object/"
