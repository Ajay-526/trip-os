export type TripStatus =
  | "draft"
  | "planning"
  | "voting"
  | "booked"
  | "in_progress"
  | "completed"
  | "cancelled";

export interface Trip {
  id: string;
  name: string;
  destination_query: string;
  start_date: string | null;
  end_date: string | null;
  budget_total_paise: number | null;
  currency: string;
  status: TripStatus;
  created_by: string;
  created_at: string;
}

export type MembershipRole = "owner" | "co_organizer" | "member";
export type MembershipStatus = "invited" | "joined" | "declined" | "removed";

export interface Membership {
  id: string;
  trip_id: string;
  user_id: string | null;
  invited_email: string | null;
  role: MembershipRole;
  status: MembershipStatus;
  joined_at: string | null;
}

export type Pace = "" | "relaxed" | "balanced" | "packed";
export type DateFlexibility = "" | "fixed" | "plus_minus_2" | "flexible";

export interface Preference {
  trip_id: string;
  user_id: string;
  budget_paise: number | null;
  pace: Pace;
  interests: string[];
  dietary: string[];
  mobility_notes: string | null;
  earliest_start_time: string | null;
  latest_end_time: string | null;
  date_flexibility: DateFlexibility;
  notes: string | null;
}

export interface PublicUser {
  id: string;
  email: string;
  display_name: string;
}

export interface ApiErrorBody {
  error: { code: string; message: string };
}
