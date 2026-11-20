package cfg

// Model definitions for AI and Image generation

// AIModel represents the available AI models for text generation
type AIModel string

// Constant names are per provider so a version bump only touches the strings here.
// Old version strings stored in the DB are rewritten on startup, see domain.legacyModels.
const (
	// AI Model identifiers (user-facing) - using actual model names
	AIModelOpenAI   AIModel = "gpt-6.2-sol"
	AIModelGemini   AIModel = "gemini-3.8-flash"
	AIModelDeepSeek AIModel = "deepseek-flash"
)

// GetAllAIModels returns all available AI model identifiers
func GetAllAIModels() []AIModel {
	return []AIModel{AIModelOpenAI, AIModelGemini, AIModelDeepSeek}
}

// IsValidAIModel checks if the given model string is a valid AI model
func IsValidAIModel(model string) bool {
	switch AIModel(model) {
	case AIModelOpenAI, AIModelGemini, AIModelDeepSeek:
		return true
	default:
		return false
	}
}

// GetAIModelBackendName returns the actual model name for the AI backend
// For most models, the user-facing name is the same as the backend name.
func GetAIModelBackendName(model AIModel) string {
	// User-facing names match backend names, so return as-is
	return string(model)
}

// ImageModel represents the available image generation models
type ImageModel string

const (
	// Image Model identifiers (user-facing) - using actual model names
	ImageModelOpenAI ImageModel = "gpt-image-2.6-sunburst"
	ImageModelBanana ImageModel = "gemini-nano-banana-2.1"
)

// GetAllImageModels returns all available image model identifiers
func GetAllImageModels() []ImageModel {
	return []ImageModel{ImageModelOpenAI, ImageModelBanana}
}

// IsValidImageModel checks if the given model string is a valid image model
func IsValidImageModel(model string) bool {
	switch ImageModel(model) {
	case ImageModelOpenAI, ImageModelBanana:
		return true
	default:
		return false
	}
}

// GetImageModelBackendName returns the actual model name for the image backend
// For most models, the user-facing name is the same as the backend name.
func GetImageModelBackendName(model ImageModel) string {
	// User-facing names match backend names, so return as-is
	return string(model)
}

// VertexAIModel returns the Vertex AI model name for Google
// ponytail: stays on gemini-3.5-flash, 3.8 on Vertex not confirmed; Vertex is only the fallback when GEMINI_DIRECT_KEY is unset
func VertexAIModel() string {
	return "gemini-3.5-flash"
}

// DefaultAIModel returns the default AI model
func DefaultAIModel() AIModel {
	return AIModelOpenAI
}

// DefaultImageModel returns the default image model
func DefaultImageModel() ImageModel {
	return ImageModelOpenAI
}
