package config

// ModelConfig holds configuration for a model.
type ModelConfig struct {
	Name           string
	DockerImage    string
	BaseURL        string
	Files          []ModelFile
	CheckpointPath string
}

// ModelFile represents a file to download for the model.
type ModelFile struct {
	Name        string
	URL         string
	Destination string
}

// FontDiffuserModel defines the configuration for FontDiffuser model.
var FontDiffuserModel = &ModelConfig{
	Name:           "fontdiffuser",
	DockerImage:    "fontdiffuser-env:latest",
	BaseURL:        "https://pub-3372efe59a304a619bc7bc0eec1c9817.r2.dev/",
	CheckpointPath: "fontdiffuser/unet/diffusion_pytorch_model.bin",
	Files: []ModelFile{
		{
			Name:        "content_encoder.pth",
			URL:         "https://pub-3372efe59a304a619bc7bc0eec1c9817.r2.dev/content_encoder.pth",
			Destination: "fontdiffuser/content_encoder.pth",
		},
		{
			Name:        "scr_210000.pth",
			URL:         "https://pub-3372efe59a304a619bc7bc0eec1c9817.r2.dev/scr_210000.pth",
			Destination: "fontdiffuser/scr_210000.pth",
		},
		{
			Name:        "style_encoder.pth",
			URL:         "https://pub-3372efe59a304a619bc7bc0eec1c9817.r2.dev/style_encoder.pth",
			Destination: "fontdiffuser/style_encoder.pth",
		},
		{
			Name:        "unet.zip",
			URL:         "https://pub-3372efe59a304a619bc7bc0eec1c9817.r2.dev/unet.zip",
			Destination: "fontdiffuser/unet.zip",
		},
	},
}

// ContentRefsURL is the URL for content references library.
const ContentRefsURL = "https://pub-3372efe59a304a619bc7bc0eec1c9817.r2.dev/content_refs.zip"
