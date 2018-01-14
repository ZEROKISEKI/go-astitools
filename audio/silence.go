package astiaudio

import (
	"math"
	"time"
)

// SilenceDetector represents a silence detector
type SilenceDetector struct {
	audioLevels *[]float64
	c           SilenceDetectorConfiguration
	samples     *[]int32
}

// SilenceDetectorConfiguration represents a silence detector configuration
type SilenceDetectorConfiguration struct {
	SilenceMinDuration time.Duration `toml:"silence_min_duration"`
	StepDuration       time.Duration `toml:"step_duration"`
}

// NewSilenceDetector creates a new silence detector
func NewSilenceDetector(c SilenceDetectorConfiguration) (d *SilenceDetector) {
	// Create
	d = &SilenceDetector{c: c}
	d.Reset()

	// Default configuration values
	if d.c.SilenceMinDuration == 0 {
		d.c.SilenceMinDuration = time.Second
	}
	if d.c.StepDuration == 0 {
		d.c.StepDuration = 30 * time.Millisecond
	}
	return
}

// Reset resets the silence detector
func (d *SilenceDetector) Reset() {
	d.audioLevels = &[]float64{}
	d.samples = &[]int32{}
}

// Add adds samples to the buffer and checks whether there are valid samples between silences
func (d *SilenceDetector) Add(samples []int32, sampleRate int, silenceMaxAudioLevel float64) (validSamples [][]int32) {
	// Append new samples
	*d.samples = append(*d.samples, samples...)

	// Get number of samples per audio level analysis
	var audioLevelAnalysisSamplesCount = int(math.Floor(float64(sampleRate) * d.c.StepDuration.Seconds()))

	// Get number of processed samples
	var processedSamplesCount = len(*d.audioLevels) * audioLevelAnalysisSamplesCount

	// Get number of processable samples
	var processableSamplesCount = len(*d.samples) - processedSamplesCount

	// Not enough processable samples
	if processableSamplesCount < audioLevelAnalysisSamplesCount {
		return
	}

	// Compute audio levels
	for i := 0; i < int(math.Floor(float64(processableSamplesCount)/float64(audioLevelAnalysisSamplesCount))); i++ {
		// Offsets
		start := processedSamplesCount + int(i*audioLevelAnalysisSamplesCount)
		end := start + audioLevelAnalysisSamplesCount

		// Append audio level
		*d.audioLevels = append(*d.audioLevels, AudioLevel((*d.samples)[start:end]))
	}

	// Count silences at the start
	var silencesCount int
	for _, l := range *d.audioLevels {
		if l < silenceMaxAudioLevel {
			silencesCount++
		} else {
			break
		}
	}

	// Keep 1 silence at the start
	if silencesCount > 1 {
		*d.audioLevels = (*d.audioLevels)[silencesCount-1:]
		*d.samples = (*d.samples)[(silencesCount-1)*audioLevelAnalysisSamplesCount:]
	}

	// Not enough audio levels to process silences in the middle
	if len(*d.audioLevels) <= 1 {
		return
	}

	// Process silences in the middle
	var i int
	silencesCount = 0
	for i = 1; i < len(*d.audioLevels); i++ {
		// Silence detected
		if (*d.audioLevels)[i] < silenceMaxAudioLevel {
			silencesCount++
			continue
		}

		// Process silences
		d.processSilencesInTheMiddle(audioLevelAnalysisSamplesCount, i, silencesCount, &validSamples)

		// Reset
		silencesCount = 0
	}

	// Process remaining silences
	d.processSilencesInTheMiddle(audioLevelAnalysisSamplesCount, i, silencesCount, &validSamples)
	return
}

// processSilencesInTheMiddle processes silences in the middle
func (d *SilenceDetector) processSilencesInTheMiddle(audioLevelAnalysisSamplesCount, i, silencesCount int, validSamples *[][]int32) {
	// Too many silences, we have valid samples!
	if time.Duration(silencesCount)*d.c.StepDuration >= d.c.SilenceMinDuration {
		// Keep 1 silence at the end
		end := (i - silencesCount) * audioLevelAnalysisSamplesCount

		// Add valid samples
		var samples = make([]int32, end)
		copy(samples, (*d.samples)[:end])
		*validSamples = append(*validSamples, samples)

		// Reset
		*d.audioLevels = (*d.audioLevels)[(i - silencesCount):]
		*d.samples = (*d.samples)[end:]
	}
}
