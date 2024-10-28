package docker

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/docker/docker/api/types/image"
)

// ListUnusedImages returns a list of unused Docker images
func (d *DockerClient) ListUnusedImages() ([]image.Summary, error) {

	images, err := d.CLI.ImageList(context.Background(), image.ListOptions{All: true})
	if err != nil {
		return nil, err
	}

	var unusedImages []image.Summary
	for _, image := range images {
		// Filter images without RepoTags (untagged or unused images)
		if len(image.RepoTags) == 0 {
			unusedImages = append(unusedImages, image)
		}
	}

	return unusedImages, nil
}

// PrintUnusedImages lists the images that would be removed (Dry Run)
func (d *DockerClient) PrintUnusedImages() error {

	images, err := d.ListUnusedImages()
	if err != nil {
		log.Printf("Error listing images: %v", err)
		return err
	}

	if len(images) == 0 {
		log.Printf("No unused images found.")
		return nil
	}

	log.Println("The following images would be removed:")
	for _, image := range images {
		log.Printf("ID: %s, Created: %d\n", image.ID, image.Created)
	}

	return nil
}

// VerboseModeCleanup gives more details while doing the cleanup of unused images
func (d *DockerClient) VerboseModeCleanup() error {

	images, err := d.ListUnusedImages()
	if err != nil {
		log.Printf("Error listing images: %v", err)
		return err
	}

	opts := image.RemoveOptions{Force: true}

	if len(images) == 0 {
		log.Println("No unused images found")
		return nil
	}

	log.Printf("Found %d unused images. Starting removal in verbose mode...\n", len(images))

	const (
		tableline   = "----------------------------------------------------------------------------------------------------------------------------------------------------"
		tableformat = "%-35s %-12s %-30s %-15s %-30s\n"
	)

	// Print table header
	fmt.Println(tableline)
	fmt.Printf(tableformat, "ID", "Size", "Created (RFC3339)", "Status", "Labels")
	fmt.Println(tableline)

	// Iterate over each unused image and attempt removal
	for _, image := range images {
		// Remove the image
		_, err := d.CLI.ImageRemove(context.Background(), image.ID, opts)
		if err != nil {
			log.Printf("Failed to remove image %s: %v\n", image.ID, err)
		} else {
			// timestamp in RFC3339 format
			created := time.Unix(image.Created, 0).Format(time.RFC3339)

			// Print image information in a table-like format
			fmt.Printf(tableformat,
				FormatDockerImageID(image.ID, 32),
				FormatSize(image.Size),
				created,
				"Removed",
				FormatLabels(image.Labels),
			)
		}
	}

	return nil
}

// Remove Images that exceed a specific size limit
func (d *DockerClient) RemoveExceedSizeLimit(sizeLimit float64, unit string) error {

	var sizeLimitInBytes int64 = int64(ToBytes(sizeLimit, unit))

	images, err := d.ListUnusedImages()
	if err != nil {
		log.Printf("Error listing images: %v", err)
		return err
	}

	if len(images) == 0 {
		log.Printf("No unused images found")
		return nil
	}

	opts := image.RemoveOptions{Force: true}

	removedImagesCount := 0
	totalSizeCleaned := int64(0)

	for _, image := range images {

		// checking and removing images exceeding the threshold size
		if image.Size > sizeLimitInBytes {
			_, err := d.CLI.ImageRemove(context.Background(), image.ID, opts)
			if err != nil {
				log.Printf("Failed to remove image %s: %v", image.ID, err)
			} else {
				log.Printf("Successfully removed image %s", image.ID)
			}

			totalSizeCleaned += image.Size
			removedImagesCount++
		}

	}

	if removedImagesCount > 0 {
		log.Printf("Summary: Removed %d images (Total space freed: %s)", removedImagesCount, FormatSize(totalSizeCleaned))
	} else {
		log.Printf("No Unused Images are exceeding the limit %d %s", int64(sizeLimit), strings.ToUpper(unit))
	}

	return nil
}

// RemoveUnusedImages deletes unused Docker images
func (d *DockerClient) RemoveUnusedImages() error {

	images, err := d.ListUnusedImages()
	if err != nil {
		log.Printf("Error listing images: %v", err)
		return err
	}

	opts := image.RemoveOptions{Force: true}

	for _, image := range images {
		_, err := d.CLI.ImageRemove(context.Background(), image.ID, opts)
		if err != nil {
			log.Printf("Failed to remove image %s: %v", image.ID, err)
		} else {
			log.Printf("Successfully removed image %s", image.ID)
		}
	}

	return nil
}
