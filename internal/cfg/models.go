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
func GetAIModelBackendName(model AIModel) string {
	switch model {
	case AIModelOAI:
		return "gpt-5"
	case AIModelDeepSeek:
		return "deepseek-v3"
	case AIModelGoogle:
		return "gemini-2.5-pro-preview"
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
func GetImageModelBackendName(model ImageModel) string {
	switch model {
	case ImageModelOAI:
		return "gpt-image-2"
	case ImageModelBanana:
		return "imagen-4.0-generate-002"
	default:
		return ""
	}
}

// VertexAIModel returns the Vertex AI model name for Google
func VertexAIModel() string {
	return "gemini-2.5-flash-001"
}

// DefaultAIModel returns the default AI model
func DefaultAIModel() AIModel {
	return AIModelOAI
}

// DefaultImageModel returns the default image model
func DefaultImageModel() ImageModel {
	return ImageModelOAI
}
