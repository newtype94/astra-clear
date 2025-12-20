import { expect } from "chai";
import { ethers } from "hardhat";
import { loadFixture } from "@nomicfoundation/hardhat-toolbox/network-helpers";
import { BankToken, Gateway, Executor } from "../typechain-types";
import { HardhatEthersSigner } from "@nomicfoundation/hardhat-ethers/signers";

/**
 * **Feature: interbank-netting-engine**
 * **Property 2: 이체 완료 시 1:1 대응**
 * **검증: 요구사항 1.5 - 소각된 금액과 발행된 금액 간의 1:1 대응 유지**
 *
 * Integration tests for the complete cross-chain transfer flow:
 * Gateway (Bank A) -> Cosmos Hub (Consensus) -> Executor (Bank B)
 */
describe("Integration: Cross-Chain Transfer Flow", function () {
  const BANK_A_CHAIN = "bank-a";
  const BANK_B_CHAIN = "bank-b";
  const INITIAL_BALANCE = ethers.parseEther("10000");

  // Helper function to sign a mint command
  async function signMintCommand(
    executor: Executor,
    commandId: string,
    recipient: string,
    amount: bigint,
    signer: HardhatEthersSigner
  ): Promise<string> {
    const messageHash = await executor.getMessageHash(commandId, recipient, amount);
    const signature = await signer.signMessage(ethers.getBytes(messageHash));
    return signature;
  }

  async function deployFullSystemFixture() {
    const [owner, validator1, validator2, validator3, userA, userB] = await ethers.getSigners();

    // Deploy Bank A Token and Gateway
    const BankToken = await ethers.getContractFactory("BankToken");
    const bankAToken = await BankToken.deploy("Bank A Token", "BNKA", owner.address);
    await bankAToken.waitForDeployment();

    const Gateway = await ethers.getContractFactory("Gateway");
    const gateway = await Gateway.deploy(
      await bankAToken.getAddress(),
      BANK_A_CHAIN,
      owner.address
    );
    await gateway.waitForDeployment();

    await bankAToken.setGateway(await gateway.getAddress());

    // Deploy Bank B Token and Executor
    const bankBToken = await BankToken.deploy("Bank B Token", "BNKB", owner.address);
    await bankBToken.waitForDeployment();

    const Executor = await ethers.getContractFactory("Executor");
    const executor = await Executor.deploy(
      await bankBToken.getAddress(),
      BANK_B_CHAIN,
      owner.address
    );
    await executor.waitForDeployment();

    await bankBToken.setExecutor(await executor.getAddress());

    // Setup validators on Executor
    await executor.updateValidatorSet([
      validator1.address,
      validator2.address,
      validator3.address,
    ]);

    // Mint initial tokens to userA on Bank A
    await bankAToken.mintInitial(userA.address, INITIAL_BALANCE);

    return {
      bankAToken,
      bankBToken,
      gateway,
      executor,
      owner,
      validator1,
      validator2,
      validator3,
      userA,
      userB,
    };
  }

  /**
   * **Property 2: 이체 완료 시 1:1 대응**
   * **Requirement 1.5: Burn amount equals mint amount**
   */
  describe("Property 2: 1:1 Correspondence", function () {
    it("Should maintain 1:1 correspondence between burned and minted tokens", async function () {
      const {
        bankAToken,
        bankBToken,
        gateway,
        executor,
        validator1,
        validator2,
        userA,
        userB,
      } = await loadFixture(deployFullSystemFixture);

      const transferAmount = ethers.parseEther("100");

      // Record initial state
      const bankATotalSupplyBefore = await bankAToken.totalSupply();
      const bankBTotalSupplyBefore = await bankBToken.totalSupply();
      const userABalanceBefore = await bankAToken.balanceOf(userA.address);
      const userBBalanceBefore = await bankBToken.balanceOf(userB.address);

      // Step 1: User A initiates transfer on Bank A (Gateway burns tokens)
      const tx = await gateway
        .connect(userA)
        .sendToChain(userB.address, transferAmount, BANK_B_CHAIN);
      const receipt = await tx.wait();

      // Extract transfer ID from event
      const transferEvent = receipt?.logs.find(
        (log) => log.topics[0] === gateway.interface.getEvent("TransferInitiated").topicHash
      );
      expect(transferEvent).to.not.be.undefined;

      // Verify Bank A supply decreased by transfer amount
      const bankATotalSupplyAfterBurn = await bankAToken.totalSupply();
      expect(bankATotalSupplyAfterBurn).to.equal(bankATotalSupplyBefore - transferAmount);

      // Verify user A balance decreased
      const userABalanceAfterBurn = await bankAToken.balanceOf(userA.address);
      expect(userABalanceAfterBurn).to.equal(userABalanceBefore - transferAmount);

      // Step 2: Simulate Cosmos Hub consensus (validators sign the mint command)
      const commandId = ethers.keccak256(ethers.toUtf8Bytes("command-1"));

      const sig1 = await signMintCommand(executor, commandId, userB.address, transferAmount, validator1);
      const sig2 = await signMintCommand(executor, commandId, userB.address, transferAmount, validator2);

      // Step 3: Execute mint on Bank B (Executor mints tokens)
      await executor.executeMint(commandId, userB.address, transferAmount, [sig1, sig2]);

      // Verify Bank B supply increased by transfer amount
      const bankBTotalSupplyAfterMint = await bankBToken.totalSupply();
      expect(bankBTotalSupplyAfterMint).to.equal(bankBTotalSupplyBefore + transferAmount);

      // Verify user B balance increased
      const userBBalanceAfterMint = await bankBToken.balanceOf(userB.address);
      expect(userBBalanceAfterMint).to.equal(userBBalanceBefore + transferAmount);

      // **Property 2 Verification: 1:1 Correspondence**
      // The amount burned on Bank A equals the amount minted on Bank B
      const burnedAmount = bankATotalSupplyBefore - bankATotalSupplyAfterBurn;
      const mintedAmount = bankBTotalSupplyAfterMint - bankBTotalSupplyBefore;
      expect(burnedAmount).to.equal(mintedAmount);
      expect(burnedAmount).to.equal(transferAmount);
    });

    it("Should maintain 1:1 correspondence across multiple transfers", async function () {
      const {
        bankAToken,
        bankBToken,
        gateway,
        executor,
        validator1,
        validator2,
        userA,
        userB,
      } = await loadFixture(deployFullSystemFixture);

      const transferAmounts = [
        ethers.parseEther("100"),
        ethers.parseEther("250"),
        ethers.parseEther("75"),
        ethers.parseEther("500"),
      ];

      let totalBurned = 0n;
      let totalMinted = 0n;

      for (let i = 0; i < transferAmounts.length; i++) {
        const amount = transferAmounts[i];

        // Burn on Bank A
        const supplyBefore = await bankAToken.totalSupply();
        await gateway.connect(userA).sendToChain(userB.address, amount, BANK_B_CHAIN);
        const supplyAfter = await bankAToken.totalSupply();
        totalBurned += supplyBefore - supplyAfter;

        // Mint on Bank B
        const commandId = ethers.keccak256(ethers.toUtf8Bytes(`command-${i + 1}`));
        const sig1 = await signMintCommand(executor, commandId, userB.address, amount, validator1);
        const sig2 = await signMintCommand(executor, commandId, userB.address, amount, validator2);

        const mintSupplyBefore = await bankBToken.totalSupply();
        await executor.executeMint(commandId, userB.address, amount, [sig1, sig2]);
        const mintSupplyAfter = await bankBToken.totalSupply();
        totalMinted += mintSupplyAfter - mintSupplyBefore;
      }

      // Verify total burned equals total minted
      expect(totalBurned).to.equal(totalMinted);

      // Verify individual amounts
      const expectedTotal = transferAmounts.reduce((a, b) => a + b, 0n);
      expect(totalBurned).to.equal(expectedTotal);
      expect(totalMinted).to.equal(expectedTotal);
    });
  });

  /**
   * **Requirement 1.3: Mint exact amount to recipient on destination chain**
   */
  describe("Requirement 1.3: Exact Amount Minting", function () {
    it("Should mint exact amount to recipient (not more, not less)", async function () {
      const { bankBToken, executor, validator1, validator2, userB } =
        await loadFixture(deployFullSystemFixture);

      const amounts = [
        ethers.parseEther("1.234567890123456789"),
        ethers.parseEther("999.999999999999999999"),
        ethers.parseEther("0.000000000000000001"), // 1 wei
      ];

      for (let i = 0; i < amounts.length; i++) {
        const amount = amounts[i];
        const balanceBefore = await bankBToken.balanceOf(userB.address);

        const commandId = ethers.keccak256(ethers.toUtf8Bytes(`exact-amount-${i}`));
        const sig1 = await signMintCommand(executor, commandId, userB.address, amount, validator1);
        const sig2 = await signMintCommand(executor, commandId, userB.address, amount, validator2);

        await executor.executeMint(commandId, userB.address, amount, [sig1, sig2]);

        const balanceAfter = await bankBToken.balanceOf(userB.address);
        expect(balanceAfter - balanceBefore).to.equal(amount);
      }
    });
  });

  /**
   * **Requirement 1.4: Recipient receives tokens within one block confirmation**
   */
  describe("Requirement 1.4: Immediate Token Receipt", function () {
    it("Should credit recipient immediately upon mint execution", async function () {
      const { bankBToken, executor, validator1, validator2, userB } =
        await loadFixture(deployFullSystemFixture);

      const amount = ethers.parseEther("100");
      const commandId = ethers.keccak256(ethers.toUtf8Bytes("immediate-receipt"));

      const balanceBefore = await bankBToken.balanceOf(userB.address);

      const sig1 = await signMintCommand(executor, commandId, userB.address, amount, validator1);
      const sig2 = await signMintCommand(executor, commandId, userB.address, amount, validator2);

      const tx = await executor.executeMint(commandId, userB.address, amount, [sig1, sig2]);
      await tx.wait();

      // Check balance immediately after transaction (same block)
      const balanceAfter = await bankBToken.balanceOf(userB.address);
      expect(balanceAfter).to.equal(balanceBefore + amount);
    });
  });

  /**
   * **Atomicity and Failure Handling**
   */
  describe("Atomicity and Failure Handling", function () {
    it("Should not mint if signature verification fails (state preserved)", async function () {
      const { bankBToken, executor, userB } = await loadFixture(deployFullSystemFixture);

      const amount = ethers.parseEther("100");
      const commandId = ethers.keccak256(ethers.toUtf8Bytes("failed-mint"));

      const supplyBefore = await bankBToken.totalSupply();
      const balanceBefore = await bankBToken.balanceOf(userB.address);

      // Try with invalid signatures (wrong signer)
      const [, , , , , nonValidator] = await ethers.getSigners();
      const invalidSig = await signMintCommand(executor, commandId, userB.address, amount, nonValidator);

      await expect(
        executor.executeMint(commandId, userB.address, amount, [invalidSig, invalidSig])
      ).to.be.revertedWith("Executor: insufficient valid signatures");

      // State should be preserved
      const supplyAfter = await bankBToken.totalSupply();
      const balanceAfter = await bankBToken.balanceOf(userB.address);

      expect(supplyAfter).to.equal(supplyBefore);
      expect(balanceAfter).to.equal(balanceBefore);
    });

    it("Should prevent double-minting with same command ID", async function () {
      const { bankBToken, executor, validator1, validator2, userB } =
        await loadFixture(deployFullSystemFixture);

      const amount = ethers.parseEther("100");
      const commandId = ethers.keccak256(ethers.toUtf8Bytes("double-mint"));

      const sig1 = await signMintCommand(executor, commandId, userB.address, amount, validator1);
      const sig2 = await signMintCommand(executor, commandId, userB.address, amount, validator2);

      // First mint should succeed
      await executor.executeMint(commandId, userB.address, amount, [sig1, sig2]);

      const balanceAfterFirst = await bankBToken.balanceOf(userB.address);

      // Second mint with same command ID should fail
      await expect(
        executor.executeMint(commandId, userB.address, amount, [sig1, sig2])
      ).to.be.revertedWith("Executor: command already processed");

      // Balance should not change
      const balanceAfterSecond = await bankBToken.balanceOf(userB.address);
      expect(balanceAfterSecond).to.equal(balanceAfterFirst);
    });
  });

  /**
   * **Property-Based: Random Amount Transfers**
   */
  describe("Property-Based: Random Transfers Maintain 1:1", function () {
    const randomAmounts = [
      ethers.parseEther("1"),
      ethers.parseEther("10"),
      ethers.parseEther("100"),
      ethers.parseEther("1000"),
      ethers.parseEther("0.001"),
      ethers.parseEther("999.123456789"),
    ];

    for (const amount of randomAmounts) {
      it(`1:1 correspondence for ${ethers.formatEther(amount)} tokens`, async function () {
        const {
          bankAToken,
          bankBToken,
          gateway,
          executor,
          validator1,
          validator2,
          userA,
          userB,
        } = await loadFixture(deployFullSystemFixture);

        // Burn
        const burnBefore = await bankAToken.totalSupply();
        await gateway.connect(userA).sendToChain(userB.address, amount, BANK_B_CHAIN);
        const burnAfter = await bankAToken.totalSupply();

        // Mint
        const commandId = ethers.keccak256(ethers.toUtf8Bytes(`random-${amount.toString()}`));
        const sig1 = await signMintCommand(executor, commandId, userB.address, amount, validator1);
        const sig2 = await signMintCommand(executor, commandId, userB.address, amount, validator2);

        const mintBefore = await bankBToken.totalSupply();
        await executor.executeMint(commandId, userB.address, amount, [sig1, sig2]);
        const mintAfter = await bankBToken.totalSupply();

        // Verify 1:1
        const burned = burnBefore - burnAfter;
        const minted = mintAfter - mintBefore;
        expect(burned).to.equal(minted);
        expect(burned).to.equal(amount);
      });
    }
  });
});
