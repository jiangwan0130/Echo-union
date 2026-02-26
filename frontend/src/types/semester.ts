export interface SemesterInfo {
  id: string;
  name: string;
  start_date: string;
  end_date: string;
  first_week_type: 'odd' | 'even';
  is_active: boolean;
  status: string;
  created_at: string;
  updated_at: string;
}

export interface SemesterBrief {
  id: string;
  name: string;
}

export interface CreateSemesterRequest {
  name: string;
  start_date: string;
  end_date: string;
  first_week_type: 'odd' | 'even';
}

export interface UpdateSemesterRequest {
  name?: string;
  start_date?: string;
  end_date?: string;
  first_week_type?: 'odd' | 'even';
  status?: 'active' | 'archived';
}
