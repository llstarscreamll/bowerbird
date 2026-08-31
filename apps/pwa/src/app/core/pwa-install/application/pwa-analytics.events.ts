export type PwaAnalyticsEvent =
  | 'pwa_visit_recorded'
  | 'pwa_install_eligible'
  | 'pwa_install_prompt_shown'
  | 'pwa_install_prompt_action'
  | 'pwa_install_native_result'
  | 'pwa_installed'
  | 'pwa_install_menu_clicked'
  | 'pwa_update_prompt_shown'
  | 'pwa_update_prompt_accepted';
