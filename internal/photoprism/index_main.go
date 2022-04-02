package photoprism

import (
	"fmt"
	"path/filepath"

	"github.com/photoprism/photoprism/internal/query"

	"github.com/photoprism/photoprism/pkg/sanitize"
)

// IndexMain indexes the main file from a group of related files and returns the result.
func IndexMain(related *RelatedFiles, ind *Index, o IndexOptions) (result IndexResult) {
	// Skip if main file is nil.
	if related.Main == nil {
		result.Err = fmt.Errorf("index: no main file for %s", sanitize.Log(related.String()))
		result.Status = IndexFailed
		return result
	}

	f := related.Main

	// Enforce file size and resolution limits.
	if exceeds, actual := f.ExceedsFileSize(o.OriginalsLimit); exceeds {
		result.Err = fmt.Errorf("index: %s exceeds file size limit (%d / %d MB)", sanitize.Log(f.RootRelName()), actual, o.OriginalsLimit)
		result.Status = IndexFailed
		return result
	} else if exceeds, actual = f.ExceedsResolution(o.ResolutionLimit); exceeds {
		result.Err = fmt.Errorf("index: %s exceeds resolution limit (%d / %d MP)", sanitize.Log(f.RootRelName()), actual, o.ResolutionLimit)
		result.Status = IndexFailed
		return result
	}

	// Extract metadata to a JSON file with Exiftool.
	if f.NeedsExifToolJson() {
		if jsonName, err := ind.convert.ToJson(f); err != nil {
			log.Debugf("index: %s in %s (extract metadata)", sanitize.Log(err.Error()), sanitize.Log(f.RootRelName()))
		} else {
			log.Debugf("index: created %s", filepath.Base(jsonName))
		}
	}

	// Create JPEG sidecar for media files in other formats so that thumbnails can be created.
	if o.Convert && f.IsMedia() && !f.HasJpeg() {
		if jpg, err := ind.convert.ToJpeg(f); err != nil {
			result.Err = fmt.Errorf("index: failed converting %s to jpeg (%s)", sanitize.Log(f.RootRelName()), err.Error())
			result.Status = IndexFailed
			return result
		} else if exceeds, actual := jpg.ExceedsResolution(o.ResolutionLimit); exceeds {
			result.Err = fmt.Errorf("index: %s exceeds resolution limit (%d / %d MP)", sanitize.Log(f.RootRelName()), actual, o.ResolutionLimit)
			result.Status = IndexFailed
			return result
		} else {
			log.Debugf("index: created %s", sanitize.Log(jpg.BaseName()))

			if err := jpg.CreateThumbnails(ind.thumbPath(), false); err != nil {
				result.Err = fmt.Errorf("index: failed creating thumbnails for %s (%s)", sanitize.Log(f.RootRelName()), err.Error())
				result.Status = IndexFailed
				return result
			}

			related.Files = append(related.Files, jpg)
		}
	}

	// Index main MediaFile.
	result = ind.MediaFile(f, o, "", "")

	// Save file error.
	if fileUid, err := result.FileError(); err != nil {
		query.SetFileError(fileUid, err.Error())
	}

	// Log index result.
	log.Infof("index: %s main %s file %s", result, f.FileType(), sanitize.Log(f.RootRelName()))

	return result
}