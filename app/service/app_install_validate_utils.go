package service

import (
	"context"
	"github.com/aihop/gopanel/app/model"
	"github.com/aihop/gopanel/buserr"
	"github.com/aihop/gopanel/constant"
)

func createLink(ctx context.Context, app model.App, appInstall *model.AppInstall, params map[string]interface{}) error {
	return nil
}
func checkRequiredAndLimit(app model.App) error {
	if err := checkLimit(app); err != nil {
		return err
	}
	return nil
}
func checkLimit(app model.App) error {
	if app.Limit > 0 {
		installs, err := appInstallRepo.ListBy(appInstallRepo.WithAppId(app.ID))
		if err != nil {
			return err
		}
		if len(installs) >= app.Limit {
			return buserr.New(constant.ErrAppLimit)
		}
	}
	return nil
}
