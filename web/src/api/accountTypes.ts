export type AccountInfo = {
    address: string;
    lastNonce: number;
    expectedNonce: number;

    confirmedBalance: number;
    pendingOut: number;
    spendableBalance: number;
};
