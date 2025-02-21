export type Wallet = {
  ID: string;
  name: string;
  role: string;
  joinedAt: Date;
};

export type User = {
  ID: string;
  name: string;
  email: string;
  pictureUrl: string;
  wallets: Wallet[];
};
