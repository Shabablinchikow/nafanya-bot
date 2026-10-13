package cfg

// Model definitions for AI and Image generation

// AIModel represents the available AI models for text generation
type AIModel string

const (
	// AI Model identifiers (user-facing) - using actual model names
	AIModelGPT55      AIModel = "gpt-5.5"
	AIModelGemini35  AIModel = "gemini-3.5-flash"
	AIModelDeepSeekV4 AIModel = "deepseek-v4-pro"
)

// GetAllAIModels returns all available AI model identifiers
func GetAllAIModels() []AIModel {
	return []AIModel{AIModelGPT55, AIModelGemini35, AIModelDeepSeekV4}
}

// IsValidAIModel checks if the given model string is a valid AI model
func IsValidAIModel(model string) bool {
	switch AIModel(model) {
	case AIModelGPT55, AIModelGemini35, AIModelDeepSeekV4:
		return true
	default:
		return false
	}
}

// GetAIModelBackendName returns the actual model name for the AI backend
// For most models, the user-facing name is the same as the backend name.
// Note: As of 2026-06-15, gemini-3.5-pro is not yet available as an API model.
// The current Pro-family baseline is gemini-3.1-pro-preview.
func GetAIModelBackendName(model AIModel) string {
	// User-facing names match backend names, so return as-is
	return string(model)
}

// ImageModel represents the available image generation models
type ImageModel string

const (
	// Image Model identifiers (user-facing) - using actual model names
	ImageModelGPTImage2    ImageModel = "gpt-image-2"
	ImageModelGemini31    ImageModel = "gemini-3.1-flash-image"
)

// GetAllImageModels returns all available image model identifiers
func GetAllImageModels() []ImageModel {
	return []ImageModel{ImageModelGPTImage2, ImageModelGemini31}
}

// IsValidImageModel checks if the given model string is a valid image model
func IsValidImageModel(model string) bool {
	switch ImageModel(model) {
	case ImageModelGPTImage2, ImageModelGemini31:
		return true
	default:
		return false
	}
}

// GetImageModelBackendName returns the actual model name for the image backend
// Note: The -preview suffixed image endpoints (gemini-3.1-flash-image-preview, gemini-3-pro-image-preview)
// are deprecated and scheduled to shut down on 2026-06-25. Using GA strings instead.
// For most models, the user-facing name is the same as the backend name.
func GetImageModelBackendName(model ImageModel) string {
	// User-facing names match backend names, so return as-is
	return string(model)
}

// VertexAIModel returns the Vertex AI model name for Google
// Using gemini-3.5-flash which is the current GA flagship-class model
func VertexAIModel() string {
	return "gemini-3.5-flash"
}

// DefaultAIModel returns the default AI model
func DefaultAIModel() AIModel {
	return AIModelGPT55
}

// DefaultImageModel returns the default image model
func DefaultImageModel() ImageModel {
	return ImageModelGPTImage2
}
