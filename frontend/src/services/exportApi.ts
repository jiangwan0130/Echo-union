import api from './api';

export const exportApi = {
  exportSchedule: (semesterId: string) =>
    api.get('/export/schedule', {
      params: { semester_id: semesterId },
      responseType: 'blob',
    }),
};
