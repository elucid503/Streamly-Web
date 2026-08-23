export interface AccessCode {

  id: string;
  code: string;
  maxUses: number;
  uses: number;

  expiresAt?: string;
  createdAt: string;

}

export interface ServiceInterruption {

  id?: string;
  enabled: boolean;
  title: string;
  message: string;
  updatedAt?: string;

}
