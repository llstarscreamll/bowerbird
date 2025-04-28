export type User = {
  ID: string;
  name: string;
  email: string;
  pictureUrl: string;
  wallets: Wallet[];
};

export type Wallet = {
  ID: string;
  name: string;
  role: string;
  joinedAt: Date;
};

export type Transaction = {
  ID: string;
  walletID: string;
  userID: string;
  userName: string;
  origin: string;
  reference: string;
  type: string;
  amount: number;
  systemDescription: string;
  userDescription: string;
  date: Date;
  processedAt: string;
  createdAt: string;
};

export type Category = {
  ID: string;
  name: string;
  color: string;
  icon: string;
};
