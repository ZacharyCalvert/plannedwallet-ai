export interface UserSession {
  username: string;
  token: string;
}

export interface UserRegistrationResponse {
  id: string;
  username: string;
  email: string;
  totp_secret: string;
  totp_url: string;
  created_at: string;
}

export interface Account {
  id: string;
  admin_id: string;
  name: string;
  type: string;
  description: string;
  created_at: string;
  updated_at: string;
}
