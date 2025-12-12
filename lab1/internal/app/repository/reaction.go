package repository

import (
	"context"
	"fmt"
	"github.com/minio/minio-go/v7"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
	"mime/multipart"
	"strings"
	"time"

	"lab1/internal/app/ds"
)

func (r *Repository) GetReactions() ([]ds.Reaction, error) {
	var reactions []ds.Reaction
	err := r.db.Where("is_delete = ?", false).Find(&reactions).Error
	if err != nil {
		return nil, err
	}
	if len(reactions) == 0 {
		return nil, fmt.Errorf("массив пустой")
	}

	return reactions, nil
}

func (r *Repository) GetReaction(id int) (ds.Reaction, error) {
	reaction := ds.Reaction{}
	err := r.db.Where("id = ? AND is_delete = ?", id, false).First(&reaction).Error
	if err != nil {
		return ds.Reaction{}, err
	}
	return reaction, nil
}

func (r *Repository) GetReactionsByTitle(title string) ([]ds.Reaction, error) {
	var reactions []ds.Reaction
	err := r.db.Where("title ILIKE ? AND is_delete = ?", "%"+title+"%", false).Find(&reactions).Error
	if err != nil {
		return nil, err
	}
	return reactions, nil
}

func (r *Repository) GetReactionsInSynthesis(creatorID uint) int64 {
	var synthesisID uint
	var count int64
	//creatorID := 1
	err := r.db.Model(&ds.Synthesis{}).Where("creator_id = ? AND status = ?", creatorID, "черновик").Select("id").First(&synthesisID).Error
	if err != nil {
		return 0
	}

	err = r.db.Model(&ds.SynthesisReaction{}).Where("synthesis_id = ?", synthesisID).Count(&count).Error
	if err != nil {
		logrus.Println("Error counting records in lists_reactions:", err)
	}

	return count
}

func (r *Repository) GetSynthesis(synthesisID uint) ([]ds.Reaction, error) {
	var reactionID uint
	var reaction ds.Reaction
	var synthesisReaction []ds.SynthesisReaction
	var result []ds.Reaction
	err := r.db.Where("synthesis_id = ?", synthesisID).Find(&synthesisReaction).Error
	if err != nil {
		return nil, err
	}

	for _, mm := range synthesisReaction {
		reactionID = mm.ReactionID
		reaction, err = r.GetReaction(int(reactionID))

		if err != nil {
			logrus.Warnf("Reaction %d not found or deleted, skipping", reactionID)
			continue
		}
		result = append(result, reaction)
	}
	return result, nil
}

func (r *Repository) FindUserSynthesis(userID uint) uint {
	var synthesisID uint
	err := r.db.Model(&ds.Synthesis{}).Where("creator_id = ? AND status = ?", userID, "черновик").Select("id").First(&synthesisID).Error
	if err != nil {
		return 0
	}
	return synthesisID
}

func (r *Repository) AddReactionInSynthesis(id uint, userID uint) error {
	//userID := r.GetUserID()
	moderatorID := r.GetModeratorID()
	var synthesisID uint
	var count int64

	err := r.db.Model(&ds.Synthesis{}).Where("creator_id = ? AND status = ?", userID, "черновик").Count(&count).Error
	if err != nil {
		return err
	}

	if count == 0 {
		newSynthesis := ds.Synthesis{
			Status:      "черновик",
			DateCreate:  time.Now(),
			DateUpdate:  time.Now(),
			CreatorID:   userID,
			ModeratorID: moderatorID,
		}
		err := r.db.Create(&newSynthesis).Error
		if err != nil {
			return err
		}
	}

	err = r.db.Model(&ds.Synthesis{}).Where("creator_id = ? AND status = ?", userID, "черновик").Select("id").First(&synthesisID).Error
	if err != nil {
		return err
	}

	var existingSynthesisReaction ds.SynthesisReaction
	err = r.db.Where("synthesis_id = ? AND reaction_id = ?", synthesisID, id).First(&existingSynthesisReaction).Error

	if err == nil {
		existingSynthesisReaction.Count++
		err = r.db.Save(&existingSynthesisReaction).Error
	} else {
		newSynthesisReaction := ds.SynthesisReaction{
			SynthesisID: synthesisID,
			ReactionID:  id,
			Count:       1,
			VolumeSM:    0,
			VolumeRM:    0,
		}
		err = r.db.Create(&newSynthesisReaction).Error
	}
	if err != nil {
		return err
	}
	return nil
}

func (r *Repository) RemoveSynthesis(id uint) error {
	deleteQuery := "UPDATE syntheses SET status = $1, date_finish = $2, date_update = $3 WHERE id = $4"
	r.db.Exec(deleteQuery, "удалён", time.Now(), time.Now(), id)
	return nil
}

func (r *Repository) GetUserNameByID(userID uint) string {
	var userName string
	r.db.Model(&ds.Users{}).Where("id = ?", userID).Select("fio").First(&userName)
	return userName
}

func (r *Repository) GetDateUpdate(synthesisID uint) string {
	var dateUpdateTime time.Time
	var dateUpdate string
	r.db.Model(&ds.Synthesis{}).Where("id = ?", synthesisID).Select("date_update").First(&dateUpdateTime)
	dateUpdate = dateUpdateTime.Format("02.01.2006 15:04:05")
	return dateUpdate
}

func (r *Repository) GetPurity(synthesisID uint) float32 {
	var purity float32
	r.db.Model(&ds.Synthesis{}).Where("id = ?", synthesisID).Select("purity").First(&purity)
	return purity
}

func (r *Repository) GetReactionCount(synthesisID uint, reactionID uint) uint {
	var synthesisReaction ds.SynthesisReaction
	err := r.db.Where("synthesis_id = ? AND reaction_id = ?", synthesisID, reactionID).First(&synthesisReaction).Error

	if err != nil {
		return 0
	}
	return synthesisReaction.Count
}

func (r *Repository) SynthesisStatusById(synthesisID uint) (string, error) {
	var SynthesisStatus string
	err := r.db.Model(&ds.Synthesis{}).Where("id = ?", synthesisID).Select("status").First(&SynthesisStatus).Error
	if err != nil {
		return "", err
	}
	return SynthesisStatus, err
}

func (r *Repository) AddReaction(reaction *ds.Reaction) error {
	if reaction.Title == "" {
		return fmt.Errorf("название реакции обязательно")
	}

	err := r.db.Model(&ds.Reaction{}).Create(map[string]interface{}{
		"title":             reaction.Title,
		"details":           reaction.Details,
		"is_delete":         reaction.IsDelete,
		"starting_material": reaction.StartingMaterial,
		"density_sm":        reaction.DensitySM,
		"molar_mass_sm":     reaction.MolarMassSM,
		"result_material":   reaction.ResultMaterial,
		"density_rm":        reaction.DensityRM,
		"molar_mass_rm":     reaction.MolarMassRM,
	}).Error
	if err != nil {
		return fmt.Errorf("ошибка при создании реакции: %w", err)
	}
	return nil
}

func (r *Repository) ChangeReaction(id uint, reactionData *ds.Reaction) error {

	var reaction ds.Reaction
	err := r.db.Where("id = ? AND is_delete = false", id).First(&reaction).Error
	if err != nil {
		return fmt.Errorf("реакция с ID %d не найдена", id)
	}

	updReaction := map[string]interface{}{
		"title":             reactionData.Title,
		"src":               reactionData.Src,
		"src_ur":            reactionData.SrcUr,
		"details":           reactionData.Details,
		"is_delete":         reactionData.IsDelete,
		"starting_material": reactionData.StartingMaterial,
		"density_sm":        reactionData.DensitySM,
		"molar_mass_sm":     reactionData.MolarMassSM,
		"result_material":   reactionData.ResultMaterial,
		"density_rm":        reactionData.DensityRM,
		"molar_mass_rm":     reactionData.MolarMassRM,
	}

	for key, value := range updReaction {
		if value == "" || value == nil {
			delete(updReaction, key)
		}
	}

	err = r.db.Model(&ds.Reaction{}).Where("id = ?", id).Updates(updReaction).Error
	if err != nil {
		return fmt.Errorf("ошибка при обновлении реакции: %w", err)
	}

	return nil
}

func (r *Repository) DeleteReaction(id uint) error {
	var reaction ds.Reaction
	err := r.db.Where("id = ?", id).First(&reaction).Error
	if err != nil {
		return fmt.Errorf("реакция с ID %d не найдена: %w", id, err)
	}

	if reaction.Src != "" {
		if err := r.DeleteReactionImage(reaction.Src); err != nil {
			logrus.Errorf("Не удалось удалить изображение для реакции %d: %v", id, err)
		}
	}
	if reaction.SrcUr != "" {
		if err := r.DeleteReactionImage(reaction.SrcUr); err != nil {
			logrus.Errorf("Не удалось удалить изображение для реакции %d: %v", id, err)
		}
	}

	err = r.db.Model(&ds.Reaction{}).Where("id = ?", id).UpdateColumn("is_delete", true).Error
	fmt.Println(id)
	if err != nil {
		return fmt.Errorf("Ошибка при удалении реакции с id %d: %w", id, err)
	}
	if err := r.CleanupDeletedReactionsFromSyntheses(); err != nil {
		logrus.Warnf("Failed to cleanup deleted reactions: %v", err)
	}
	return nil
}

func (r *Repository) DeleteReactionImage(src string) error {
	if src == "" {
		logrus.Info("Empty image source, skipping deletion")
		return nil
	}
	objectName := r.extractObjectName(src)
	logrus.Infof("Extracted object name: '%s' from source: '%s'", objectName, src)
	if objectName == "" {
		logrus.Warnf("Could not extract object name from src: %s", src)
		return nil
	}
	logrus.Infof("Attempting to delete object: '%s' from bucket: '%s'", objectName, r.bucketName)

	_, err := r.minioClient.StatObject(context.Background(), r.bucketName, objectName, minio.StatObjectOptions{})
	if err != nil {
		logrus.Warnf("Object '%s' not found in MinIO: %v", objectName, err)
		return nil
	}

	err = r.minioClient.RemoveObject(context.Background(), r.bucketName, objectName, minio.RemoveObjectOptions{})
	if err != nil {
		logrus.Errorf("Failed to delete object '%s' from MinIO: %v", objectName, err)
		return fmt.Errorf("ошибка при удалении изображения из MinIO: %w", err)
	}

	logrus.Infof("Successfully deleted object: '%s' from bucket: '%s'", objectName, r.bucketName)
	return nil
}

func (r *Repository) extractObjectName(src string) string {
	logrus.Infof("Processing source URL: %s", src)
	if strings.Contains(src, "?") {
		src = strings.Split(src, "?")[0]
	}
	if strings.Contains(src, "http://localhost:9000/") {
		parts := strings.Split(src, "/")
		for i, part := range parts {
			if part == r.bucketName && i+1 < len(parts) {
				objectPath := strings.Join(parts[i+1:], "/")
				logrus.Infof("Extracted object path: %s", objectPath)
				return objectPath
			}
		}
	}
	if !strings.Contains(src, "/") {
		return "img/" + src
	}
	logrus.Infof("Using as-is: %s", src)
	return src
}

func (r *Repository) CleanupDeletedReactionsFromSyntheses() error {
	var deletedReactions []ds.Reaction
	err := r.db.Where("is_delete = ?", true).Find(&deletedReactions).Error
	if err != nil {
		return err
	}

	for _, reaction := range deletedReactions {
		err = r.db.Where("reaction_id = ?", reaction.ID).Delete(&ds.SynthesisReaction{}).Error
		if err != nil {
			logrus.Warnf("Failed to remove reaction %d from syntheses: %v", reaction.ID, err)
		} else {
			logrus.Infof("Removed deleted reaction %d from all syntheses", reaction.ID)
		}
	}
	return nil
}

func (r *Repository) UploadReactionImage(id uint, fileHeader *multipart.FileHeader) error {
	var reaction ds.Reaction
	err := r.db.Where("id = ? AND is_delete = false", id).First(&reaction).Error
	if err != nil {
		return fmt.Errorf("реакция с ID %d не найдена", id)
	}

	if reaction.Src != "" {
		if err := r.DeleteReactionImage(reaction.Src); err != nil {
			logrus.Errorf("Не удалось удалить старое изображение: %v", err)
		}
	}

	fileName := fmt.Sprintf("img/reaction_%d_%s", id, fileHeader.Filename)

	file, err := fileHeader.Open()
	if err != nil {
		return fmt.Errorf("ошибка открытия файла: %w", err)
	}
	defer file.Close()

	_, err = r.minioClient.PutObject(
		context.Background(),
		r.bucketName,
		fileName,
		file,
		fileHeader.Size,
		minio.PutObjectOptions{
			ContentType: fileHeader.Header.Get("Content-Type"),
		},
	)
	if err != nil {
		return fmt.Errorf("ошибка загрузки в MinIO: %w", err)
	}

	reaction.Src = "http://localhost:9000/aspirinimages/" + fileName
	err = r.db.Save(&reaction).Error
	if err != nil {
		r.minioClient.RemoveObject(context.Background(), r.bucketName, fileName, minio.RemoveObjectOptions{})
		return fmt.Errorf("ошибка сохранения пути к изображению: %w", err)
	}

	return nil
}

type SynthesisReactionWithCount struct {
	ds.Reaction
	VolumeSM float32 `json:"volume_sm"`
	VolumeRM float32 `json:"volume_rm,omitempty"`
	//SynthesisReactionID uint    `json:"synthesis_reaction_id"`
	Count uint
}

func (r *Repository) GetSynthesisWithCounts(synthesisID uint) ([]SynthesisReactionWithCount, error) {
	var synthesisReactions []ds.SynthesisReaction
	var result []SynthesisReactionWithCount

	err := r.db.Where("synthesis_id = ?", synthesisID).Find(&synthesisReactions).Error
	if err != nil {
		return nil, err
	}

	for _, sr := range synthesisReactions {
		reaction, err := r.GetReaction(int(sr.ReactionID))
		if err != nil {
			logrus.Warnf("Reaction %d not found or deleted, skipping", sr.ReactionID)
			continue
		}

		result = append(result, SynthesisReactionWithCount{
			Reaction: reaction,
			Count:    sr.Count,
			VolumeSM: sr.VolumeSM,
			VolumeRM: sr.VolumeRM,
		})
	}
	return result, nil
}

func (r *Repository) GetSynthesisCount(creatorID uint) int64 {
	var synthesisID uint
	var count int64
	//creatorID := 1
	err := r.db.Model(&ds.Synthesis{}).Where("creator_id = ? AND status = ?", creatorID, "черновик").Select("id").First(&synthesisID).Error
	if err != nil {
		return 0
	}

	err = r.db.Model(&ds.SynthesisReaction{}).Where("synthesis_id = ?", synthesisID).Count(&count).Error
	if err != nil {
		logrus.Println("Error counting records in list_chats:", err)
	}

	return count
}

func (r *Repository) GetSynthesisID(userID uint) int {
	var synthesisID int
	err := r.db.Model(&ds.Synthesis{}).Where("creator_id = ? AND status = ?", userID, "черновик").Select("id").First(&synthesisID).Error
	if err != nil {
		return 0
	}
	return synthesisID
}

func (r *Repository) GetSyntheses(status, startDate, endDate string, userId uint) ([]ds.Synthesis, error) {
	var synthesis []ds.Synthesis
	var isModerator bool
	r.db.Model(&ds.Users{}).Where("id = ?", userId).Select("is_moderator").First(&isModerator)
	fmt.Println("Параметры фильтрации:", status, startDate, endDate)
	var query *gorm.DB
	if isModerator {
		query = r.db.Where("status != ? AND status != ?", "удалён", "черновик")
	} else {
		query = r.db.Where("status != ? AND status != ? AND creator_id = ?", "удалён", "черновик", userId)
	}

	if status != "" {
		query = query.Where("status = ?", status)
	}

	if startDate != "" {
		start, err := time.Parse("2006-01-02", startDate)
		if err == nil {
			query = query.Where("date_create >= ?", start)
		} else {
			fmt.Println("Ошибка парсинга startDate:", err)
		}
	}
	if endDate != "" {
		end, err := time.Parse("2006-01-02", endDate)
		if err == nil {
			query = query.Where("date_create <= ?", end.AddDate(0, 0, 1))
		} else {
			fmt.Println("Ошибка парсинга endDate:", err)
		}
	}

	err := query.Preload("Creator").Preload("Moderator").Find(&synthesis).Error
	if err != nil {
		return nil, fmt.Errorf("ошибка получения синтеза: %w", err)
	}
	return synthesis, nil
}

func (r *Repository) GetSynthesisByID(synthesisID uint) (*ds.Synthesis, []ds.Reaction, error) {
	var synthesis ds.Synthesis

	err := r.db.
		Preload("Creator").
		Preload("Moderator").
		Where("id = ?", synthesisID).
		First(&synthesis).Error

	if err != nil {
		return nil, nil, fmt.Errorf("синтез с ID %d не найдена", synthesisID)
	}

	var reactions []ds.Reaction
	err = r.db.
		Table("reactions").
		Joins("JOIN synthesis_reactions ON reactions.id = synthesis_reactions.reaction_id").
		Where("synthesis_reactions.synthesis_id = ?", synthesisID).
		Find(&reactions).Error

	if err != nil {
		return nil, nil, fmt.Errorf("ошибка загрузки реакции: %w", err)
	}

	return &synthesis, reactions, nil
}

func (r *Repository) UpdateSynthesisPurity(synthesisID uint, Purity float64) error {
	var synthesis ds.Synthesis
	err := r.db.Where("id = ? AND status = ?", synthesisID, "черновик").First(&synthesis).Error
	if err != nil {
		return fmt.Errorf("синтез-черновик с ID %d не найдена", synthesisID)
	}

	if Purity > 0 && Purity <= 100 {
		err = r.db.Model(&ds.Synthesis{}).Where("id = ?", synthesisID).Updates(map[string]interface{}{
			"purity":      Purity,
			"date_update": time.Now(),
		}).Error
		if err != nil {
			return fmt.Errorf("ошибка при обновлении синтеза: %w", err)
		}
	} else {
		return fmt.Errorf("Неккоректный ввод концентрации  Концентрация должна быть больше 0 и не больше 100")
	}

	return nil
}

func (r *Repository) FormSynthesis(synthesisID uint) error {
	var synthesis ds.Synthesis
	err := r.db.Model(&ds.Synthesis{}).Where("id = ? AND status = ?", synthesisID, "черновик").
		First(&synthesis).Error

	if err != nil {
		return fmt.Errorf("синтез-черновик с ID %d не найден", synthesisID)
	}

	if synthesis.Purity <= 0 || synthesis.Purity > 100 {
		return fmt.Errorf("поле Purity отсутствует или не соответствует требованиям")
	}

	var countReactionsAtSynthesis int64
	err = r.db.Model(&ds.SynthesisReaction{}).Where("synthesis_id = ?", synthesisID).Count(&countReactionsAtSynthesis).Error

	if countReactionsAtSynthesis == 0 {
		return fmt.Errorf("добавьте хотя бы одну реакцию для формирования синтеза")
	}

	err = r.db.Model(&ds.Synthesis{}).Where("id = ?", synthesisID).Updates(map[string]interface{}{
		"status":      "сформирован",
		"date_update": time.Now(),
	}).Error

	if err != nil {
		return fmt.Errorf("ошибка при формировании синтеза: %w", err)
	}
	return nil
}

//	func (r *Repository) CompleteOrRejectSynthesis(synthesisID uint, moderatorID uint, newStatus bool) error {
//		var synthesis ds.Synthesis
//		err := r.db.Model(&ds.Synthesis{}).Where("id = ? AND status = ?", synthesisID, "сформирован").First(&synthesis).Error
//
//		if err != nil {
//			return fmt.Errorf("сформированный синтез с ID %d не найден", synthesisID)
//		}
//		var updStatus string
//		if newStatus {
//			updStatus = "завершён"
//		} else {
//			updStatus = "отклонён"
//		}
//		updates := map[string]interface{}{
//			"status":       updStatus,
//			"date_update":  time.Now(),
//			"date_finish":  time.Now(),
//			"moderator_id": moderatorID,
//		}
//
//		if newStatus {
//			var synthesisReaction []ds.SynthesisReaction
//			err = r.db.Where("synthesis_id = ?", synthesisID).Find(&synthesisReaction).Error
//			if err != nil {
//				return fmt.Errorf("ошибка получения данных о синтезе: %w", err)
//			}
//
//			for _, reactionFromSynthesis := range synthesisReaction {
//				var reaction ds.Reaction
//				var res float32
//				err = r.db.Where("id = ?", reactionFromSynthesis.ReactionID).Find(&reaction).Error
//				if err != nil {
//					return fmt.Errorf("ошибка получения данных о синтезе: %w", err)
//				}
//				// Vk = (c*Vs*ps*Mk)/(pk*Ms)
//				purity := synthesis.Purity
//				volumeSM := reactionFromSynthesis.VolumeSM
//				molarMassSM := float32(reaction.MolarMassSM)
//				molarMassRM := float32(reaction.MolarMassRM)
//				count := float32(reactionFromSynthesis.Count)
//				densitySM := reaction.DensitySM
//				densityRM := reaction.DensityRM
//				res = ((purity * volumeSM * densitySM * molarMassRM) / (densityRM * molarMassSM)) * count
//
//				r.db.Model(&ds.SynthesisReaction{}).Where("id = ?", reactionFromSynthesis.ID).Update("volume_rm", res)
//			}
//		}
//
//		err = r.db.Model(&ds.Synthesis{}).Where("id = ?", synthesisID).Updates(updates).Error
//		if err != nil {
//			return fmt.Errorf("ошибка при обновлении синтеза: %w", err)
//		}
//
//		return nil
//	}
func (r *Repository) CompleteOrRejectSynthesis(synthesisID uint, moderatorID uint, newStatus bool) error {
	var synthesis ds.Synthesis
	err := r.db.Model(&ds.Synthesis{}).Where("id = ? AND status = ?", synthesisID, "сформирован").First(&synthesis).Error

	if err != nil {
		return fmt.Errorf("сформированный синтез с ID %d не найден", synthesisID)
	}

	var updStatus string
	if newStatus {
		updStatus = "завершён"
	} else {
		updStatus = "отклонён"
	}

	updates := map[string]interface{}{
		"status":       updStatus,
		"date_update":  time.Now(),
		"date_finish":  time.Now(),
		"moderator_id": moderatorID,
	}

	// Убираем синхронный расчёт
	// if newStatus { ... }

	err = r.db.Model(&ds.Synthesis{}).Where("id = ?", synthesisID).Updates(updates).Error
	if err != nil {
		return fmt.Errorf("ошибка при обновлении синтеза: %w", err)
	}

	return nil
}

func (r *Repository) DeleteSynthesis(synthesisID uint) error {
	var synthesis ds.Synthesis
	err := r.db.Where("id = ?", synthesisID).First(&synthesis).Error
	if err != nil {
		return fmt.Errorf("синтез с ID %d не найдена", synthesisID)
	}

	err = r.db.Model(&ds.Synthesis{}).Where("id = ?", synthesisID).Updates(map[string]interface{}{
		"status":      "удалён",
		"date_update": time.Now(),
	}).Error

	if err != nil {
		return fmt.Errorf("ошибка при удалении синтеза: %w", err)
	}

	return nil
}

func (r *Repository) RemoveReactionFromSynthesis(synthesisID uint, reactionID uint) error {
	var synthesis ds.Synthesis
	err := r.db.Where("id = ? AND status = ?", synthesisID, "черновик").First(&synthesis).Error
	if err != nil {
		return fmt.Errorf("синтез-черновик с ID %d не найден", synthesisID)
	}

	var reaction ds.Reaction
	err = r.db.Where("id = ? AND is_delete = false", reactionID).First(&reaction).Error
	if err != nil {
		return fmt.Errorf("реакция с ID %d не найдена", reactionID)
	}
	var count int64
	err = r.db.Model(&ds.SynthesisReaction{}).Where("synthesis_id = ? AND reaction_id = ?", synthesisID, reactionID).Count(&count).Error
	if err != nil || count == 0 {
		return fmt.Errorf("реакция с ID %d в синтезе с ID %d не найдена", reactionID, synthesisID)
	}
	count = 0
	err = r.db.Model(&ds.SynthesisReaction{}).Where("synthesis_id = ? AND reaction_id = ?", synthesisID, reactionID).Select("count").First(&count).Error
	count -= 1
	if count == 0 {
		err = r.db.Where("synthesis_id = ? AND reaction_id = ?", synthesisID, reactionID).Delete(&ds.SynthesisReaction{}).Error
		if err != nil {
			return fmt.Errorf("ошибка при удалении реакции из синтеза: %w", err)
		}
	} else {
		err = r.db.Model(&ds.SynthesisReaction{}).Where("synthesis_id = ? AND reaction_id = ?", synthesisID, reactionID).UpdateColumn("count", uint(count)).Error
		if err != nil {
			return fmt.Errorf("ошибка при удалении реакции из синтеза: %w", err)
		}
	}
	return nil
}

func (r *Repository) UpdateReactionInSynthesis(synthesisID uint, reactionID uint, volume float64) error {
	var synthesis ds.Synthesis
	err := r.db.Where("id = ? AND status = ?", synthesisID, "черновик").First(&synthesis).Error
	if err != nil {
		return fmt.Errorf("синтез-черновик с ID %d не найден", synthesisID)
	}

	var reaction ds.Reaction
	err = r.db.Where("id = ? AND is_delete = false", reactionID).First(&reaction).Error
	if err != nil {
		return fmt.Errorf("реакция с ID %d не найдена", reactionID)
	}

	var synthesisReaction ds.SynthesisReaction
	err = r.db.Where("synthesis_id = ? AND reaction_id = ?", synthesisID, reactionID).First(&synthesisReaction).Error
	if err != nil {
		return fmt.Errorf("реакция не найдена в синтезе")
	}

	err = r.db.Model(&ds.SynthesisReaction{}).
		Where("synthesis_id = ? AND reaction_id = ?", synthesisID, reactionID).
		Update("volume_sm", volume).Error

	if err != nil {
		return fmt.Errorf("ошибка при обновлении реакции в синтезе: %w", err)
	}

	return nil
}

func (r *Repository) RegisterUser(login, password, fio string) error {
	if login == "" {
		return fmt.Errorf("логин не может быть пустым")
	}
	if password == "" {
		return fmt.Errorf("пароль не может быть пустым")
	}
	var existingUser ds.Users
	err := r.db.Where("login = ?", login).First(&existingUser).Error
	if err == nil {
		return fmt.Errorf("логином '%s' уже занят", login)
	}

	newUser := ds.Users{
		Login:       login,
		Password:    password,
		IsModerator: false,
		FIO:         fio,
	}

	err = r.db.Create(&newUser).Error
	if err != nil {
		return fmt.Errorf("ошибка при создании пользователя: %w", err)
	}
	//newUser.Password = ""
	return err
}

func (r *Repository) GetUserProfile(userID uint) (*ds.Users, error) {
	var user ds.Users

	err := r.db.Where("id = ?", userID).First(&user).Error
	if err != nil {
		return nil, fmt.Errorf("пользователь не найден")
	}
	user.Password = ""
	return &user, nil
}

func (r *Repository) AuthenticateUser(login, password string) (*ds.Users, error) {
	var user ds.Users
	err := r.db.Where("login = ?", login).First(&user).Error
	if err != nil {
		return nil, fmt.Errorf("неверный логин или пароль")
	}
	if user.Password != password {
		return nil, fmt.Errorf("неверный логин или пароль")
	}
	user.Password = ""
	return &user, nil
}

func (r *Repository) AuthUser(login, password string) (*ds.Users, error) {
	var user ds.Users
	err := r.db.Where("login = ?", login).First(&user).Error
	if err != nil {
		return nil, fmt.Errorf("неверный логин или пароль")
	}
	if user.Password != password {
		return nil, fmt.Errorf("неверный логин или пароль")
	}
	user.Password = ""
	return &user, nil
}

func (r *Repository) UpdateUser(userID uint, updates map[string]interface{}) (*ds.Users, error) {
	if login, exists := updates["login"]; exists && login != "" {
		var existingUser ds.Users
		err := r.db.Where("login = ? AND id != ?", login, userID).First(&existingUser).Error
		if err == nil {
			return nil, fmt.Errorf("логин '%s' уже занят", login)
		}
	}
	if len(updates) > 0 {
		err := r.db.Model(&ds.Users{}).Where("id = ?", userID).Updates(updates).Error
		if err != nil {
			return nil, fmt.Errorf("ошибка обновления: %w", err)
		}
	}
	var user ds.Users
	r.db.Where("id = ?", userID).First(&user)
	user.Password = ""

	return &user, nil
}

func (r *Repository) GetUserByLogin(login string) (*ds.Users, error) {
	user := &ds.Users{
		Login: "login",
	}

	err := r.db.First(user).Error
	if err != nil {
		return nil, err
	}

	return user, nil
}

func (r *Repository) GetUserID() uint {
	return 1
}

func (r *Repository) GetModeratorID() uint {
	return 2
}

func (r *Repository) UpdateReactionVolumeRM(synthesisID, reactionID uint, volumeRM float32) error {
	logrus.Infof("DB Update: synthesis_id=%d, reaction_id=%d, volume_rm=%f",
		synthesisID, reactionID, volumeRM)

	result := r.db.Model(&ds.SynthesisReaction{}).
		Where("synthesis_id = ? AND reaction_id = ?", synthesisID, reactionID).
		Update("volume_rm", volumeRM)

	logrus.Infof("Rows affected: %d", result.RowsAffected)
	logrus.Infof("DB Error: %v", result.Error)

	return result.Error
}

//	func (r *Repository) GetSynthesisDataForCalculation(synthesisID uint) ([]ds.SynthesisReaction, error) {
//		var synthesisReactions []ds.SynthesisReaction
//		err := r.db.Where("synthesis_id = ?", synthesisID).
//			Preload("Reaction").
//			Find(&synthesisReactions).Error
//		if err != nil {
//			return nil, fmt.Errorf("ошибка получения данных синтеза: %w", err)
//		}
//		return synthesisReactions, nil
//	}
//
// В repository/repository.go
func (r *Repository) GetSynthesisDataForCalculation(synthesisID uint) ([]SynthesisReactionWithDetails, error) {
	var synthesisReactions []SynthesisReactionWithDetails

	err := r.db.Table("synthesis_reactions sr").
		Select(`
            sr.reaction_id,
            sr.volume_sm,
            sr.count,
            s.purity,
            r.density_sm,
            r.molar_mass_sm,
            r.density_rm,
            r.molar_mass_rm
        `).
		Joins("JOIN syntheses s ON s.id = sr.synthesis_id").
		Joins("JOIN reactions r ON r.id = sr.reaction_id").
		Where("sr.synthesis_id = ?", synthesisID).
		Scan(&synthesisReactions).Error

	if err != nil {
		return nil, fmt.Errorf("ошибка получения данных синтеза: %w", err)
	}

	return synthesisReactions, nil
}

type SynthesisReactionWithDetails struct {
	ReactionID  uint    `json:"reaction_id"`
	VolumeSM    float32 `json:"volume_sm"`
	Count       uint    `json:"count"`
	Purity      float32 `json:"purity"`
	DensitySM   float32 `json:"density_sm"`
	MolarMassSM int     `json:"molar_mass_sm"`
	DensityRM   float32 `json:"density_rm"`
	MolarMassRM int     `json:"molar_mass_rm"`
}
