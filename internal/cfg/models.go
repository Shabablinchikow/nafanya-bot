package cfg

// Model definitions for AI and Image generation

// AIModel represents the available AI models for text generation
type AIModel string

const (
	// AI Model identifiers (user-facing)
	AIModelOAI      AIModel = "oai"
	AIModelGoogle   AIModel = "google"
	AIModelDeepSeek AIModel = "deepseek"
)

// GetAllAIModels returns all available AI model identifiers
func GetAllAIModels() []AIModel {
	return []AIModel{AIModelOAI, AIModelGoogle, AIModelDeepSeek}
}

// IsValidAIModel checks if the given model string is a valid AI model
func IsValidAIModel(model string) bool {
	switch AIModel(model) {
	case AIModelOAI, AIModelGoogle, AIModelDeepSeek:
		return true
	default:
		return false
	}
}

// GetAIModelBackendName returns the actual model name for the AI backend
// Note: As of 2026-06-15, gemini-3.5-pro is not yet available as an API model.
// The current Pro-family baseline is gemini-3.1-pro-preview.
func GetAIModelBackendName(model AIModel) string {
	switch model {
	case AIModelOAI:
		return "gpt-5.5"
	case AIModelDeepSeek:
		return "deepseek-v4-pro"
	case AIModelGoogle:
		return "gemini-3.5-flash"
	default:
		return ""
	}
}

// ImageModel represents the available image generation models
type ImageModel string

const (
	// Image Model identifiers (user-facing)
	ImageModelOAI    ImageModel = "oai"
	ImageModelBanana ImageModel = "banana"
)

// GetAllImageModels returns all available image model identifiers
func GetAllImageModels() []ImageModel {
	return []ImageModel{ImageModelOAI, ImageModelBanana}
}

// IsValidImageModel checks if the given model string is a valid image model
func IsValidImageModel(model string) bool {
	switch ImageModel(model) {
	case ImageModelOAI, ImageModelBanana:
		return true
	default:
		return false
	}
}

// GetImageModelBackendName returns the actual model name for the image backend
// Note: The -preview suffixed image endpoints (gemini-3.1-flash-image-preview, gemini-3-pro-image-preview)
// are deprecated and scheduled to shut down on 2026-06-25. Using GA strings instead.
func GetImageModelBackendName(model ImageModel) string {
	switch model {
	case ImageModelOAI:
		return "gpt-image-2"
	case ImageModelBanana:
		return "gemini-3.1-flash-image"
	default:
		return ""
	}
}

// VertexAIModel returns the Vertex AI model name for Google
// Using gemini-3.5-flash which is the current GA flagship-class model
func VertexAIModel() string {
	return "gemini-3.5-flash"
}

// DefaultAIModel returns the default AI model
func DefaultAIModel() AIModel {
	return AIModelOAI
}

// DefaultImageModel returns the default image model
func DefaultImageModel() ImageModel {
	return ImageModelOAI
}
