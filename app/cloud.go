// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package app

import (
	"net/http"

	"github.com/mattermost/mattermost-server/v5/mlog"
	"github.com/mattermost/mattermost-server/v5/model"
)

func (a *App) getSysAdminsEmailRecipients() ([]*model.User, *model.AppError) {
	userOptions := &model.UserGetOptions{
		Page:     0,
		PerPage:  100,
		Role:     model.SYSTEM_ADMIN_ROLE_ID,
		Inactive: false,
	}
	return a.GetUsers(userOptions)
}

func (a *App) CheckAndSendUserLimitWarningEmails() *model.AppError {
	if a.Srv().License() == nil || (a.Srv().License() != nil && !*a.Srv().License().Features.Cloud) {
		// Not cloud instance, do nothing
		return nil
	}

	subscription, err := a.Cloud().GetSubscription(a.Session().UserId)
	if err != nil {
		return model.NewAppError(
			"app.CheckAndSendUserLimitWarningEmails",
			"api.cloud.get_subscription.error",
			nil,
			err.Error(),
			http.StatusInternalServerError)
	}

	if subscription != nil && subscription.IsPaidTier == "true" {
		// Paid subscription, do nothing
		return nil
	}

	cloudUserLimit := *a.Config().ExperimentalSettings.CloudUserLimit
	systemUserCount, _ := a.Srv().Store.User().Count(model.UserCountOptions{})
	remainingUsers := cloudUserLimit - systemUserCount

	if remainingUsers > 0 {
		return nil
	}
	sysAdmins, appErr := a.getSysAdminsEmailRecipients()
	if appErr != nil {
		return model.NewAppError(
			"app.CheckAndSendUserLimitWarningEmails",
			"api.cloud.get_admins_emails.error",
			nil,
			appErr.Error(),
			http.StatusInternalServerError)
	}

	// -1 means they are 1 user over the limit - we only want to send the email for the 11th user
	if remainingUsers == -1 {
		// Over limit by 1 user
		for admin := range sysAdmins {
			_, appErr := a.Srv().EmailService.SendOverUserLimitWarningEmail(sysAdmins[admin].Email, sysAdmins[admin].Locale, *a.Config().ServiceSettings.SiteURL)
			if appErr != nil {
				a.Log().Error(
					"Error sending user limit warning email to admin",
					mlog.String("username", sysAdmins[admin].Username),
					mlog.Err(err),
				)
			}
		}
	} else if remainingUsers == 0 {
		// At limit
		for admin := range sysAdmins {
			_, appErr := a.Srv().EmailService.SendAtUserLimitWarningEmail(sysAdmins[admin].Email, sysAdmins[admin].Locale, *a.Config().ServiceSettings.SiteURL)
			if appErr != nil {
				a.Log().Error(
					"Error sending user limit warning email to admin",
					mlog.String("username", sysAdmins[admin].Username),
					mlog.Err(err),
				)
			}
		}
	}
	return nil
}

func (a *App) SendPaymentFailedEmail(failedPayment *model.FailedPayment) *model.AppError {
	sysAdmins, err := a.getSysAdminsEmailRecipients()
	if err != nil {
		return err
	}

	for _, admin := range sysAdmins {
		_, err := a.Srv().EmailService.SendPaymentFailedEmail(admin.Email, admin.Locale, failedPayment, *a.Config().ServiceSettings.SiteURL)
		if err != nil {
			a.Log().Error("Error sending payment failed email", mlog.Err(err))
		}
	}
	return nil
}

// SendNoCardPaymentFailedEmail
func (a *App) SendNoCardPaymentFailedEmail() *model.AppError {
	sysAdmins, err := a.getSysAdminsEmailRecipients()
	if err != nil {
		return err
	}

	for _, admin := range sysAdmins {
		err := a.Srv().EmailService.SendNoCardPaymentFailedEmail(admin.Email, admin.Locale, *a.Config().ServiceSettings.SiteURL)
		if err != nil {
			a.Log().Error("Error sending payment failed email", mlog.Err(err))
		}
	}
	return nil
}
