// Package jobs executa tarefas de reconciliação de dados
package jobs

import (
	"SafeBox/repositories"
	"SafeBox/services/storage"
	"context"
	"log"
	"time"
)

func StartReconciliationJob(
	repo repositories.QuotaRepositoryInterface,
	storage storage.StorageRepository,
) {
	ticker := time.NewTicker(24 * time.Hour)
	defer ticker.Stop()

	for range ticker.C {
		log.Println("[JOB] Iniciando reconciliação de cotas...")

		ctx := context.Background()
		users, err := repo.GetAllUsers(ctx)
		if err != nil {
			log.Printf("[JOB] Erro ao obter usuários: %v", err)
			continue
		}

		for _, user := range users {
			actualUsed, err := storage.GetTotalUsage(ctx, user.UserID)
			if err != nil {
				log.Printf("[JOB] Erro ao calcular uso para usuário %d: %v", user.UserID, err)
				continue
			}

			if err := repo.UpdateUsage(ctx, user.UserID, actualUsed); err != nil {
				log.Printf("[JOB] Erro ao atualizar cota do usuário %d: %v", user.UserID, err)
			}
		}

		log.Println("[JOB] Reconciliação concluída com sucesso")
	}
}
