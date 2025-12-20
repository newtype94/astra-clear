import { expect } from "chai";
import { ethers } from "hardhat";
import { loadFixture } from "@nomicfoundation/hardhat-toolbox/network-helpers";
import { BankToken, Gateway } from "../typechain-types";
import { HardhatEthersSigner } from "@nomicfoundation/hardhat-ethers/signers";

/**
 * **Feature: interbank-netting-engine, Property 1: 토큰 소각 및 이벤트 발생**
 * **검증: 요구사항 1.1, 1.2**
 */
describe("Gateway", function () {
  const SOURCE_CHAIN = "bank-a";
  const DEST_CHAIN = "bank-b";
  const INITIAL_BALANCE = ethers.parseEther("10000");

  async function deployGatewayFixture() {
    const [owner, user1, user2] = await ethers.getSigners();

    // Deploy BankToken
    const BankToken = await ethers.getContractFactory("BankToken");
    const token = await BankToken.deploy("Bank A Token", "BNKA", owner.address);
    await token.waitForDeployment();

    // Deploy Gateway
    const Gateway = await ethers.getContractFactory("Gateway");
    const gateway = await Gateway.deploy(
      await token.getAddress(),
      SOURCE_CHAIN,
      owner.address
    );
    await gateway.waitForDeployment();

    // Set gateway in token
    await token.setGateway(await gateway.getAddress());

    // Mint initial tokens to user1
    await token.mintInitial(user1.address, INITIAL_BALANCE);

    return { token, gateway, owner, user1, user2 };
  }

  describe("Deployment", function () {
    it("Should set the correct token address", async function () {
      const { token, gateway } = await loadFixture(deployGatewayFixture);
      expect(await gateway.token()).to.equal(await token.getAddress());
    });

    it("Should set the correct source chain", async function () {
      const { gateway } = await loadFixture(deployGatewayFixture);
      expect(await gateway.sourceChain()).to.equal(SOURCE_CHAIN);
    });

    it("Should start with nonce 0", async function () {
      const { gateway } = await loadFixture(deployGatewayFixture);
      expect(await gateway.getCurrentNonce()).to.equal(0);
    });
  });

  /**
   * **Property 1: 토큰 소각 및 이벤트 발생**
   * **Requirement 1.1: Burns tokens immediately from sender**
   */
  describe("Property 1: Token Burn", function () {
    it("Should burn tokens from sender when sendToChain is called", async function () {
      const { token, gateway, user1 } = await loadFixture(deployGatewayFixture);
      const amount = ethers.parseEther("100");
      const recipient = "cosmos1recipient...";

      const balanceBefore = await token.balanceOf(user1.address);

      await gateway.connect(user1).sendToChain(recipient, amount, DEST_CHAIN);

      const balanceAfter = await token.balanceOf(user1.address);
      expect(balanceAfter).to.equal(balanceBefore - amount);
    });

    it("Should reduce total supply when tokens are burned", async function () {
      const { token, gateway, user1 } = await loadFixture(deployGatewayFixture);
      const amount = ethers.parseEther("100");
      const recipient = "cosmos1recipient...";

      const supplyBefore = await token.totalSupply();

      await gateway.connect(user1).sendToChain(recipient, amount, DEST_CHAIN);

      const supplyAfter = await token.totalSupply();
      expect(supplyAfter).to.equal(supplyBefore - amount);
    });

    it("Should fail if sender has insufficient balance", async function () {
      const { gateway, user2 } = await loadFixture(deployGatewayFixture);
      const amount = ethers.parseEther("100");
      const recipient = "cosmos1recipient...";

      await expect(
        gateway.connect(user2).sendToChain(recipient, amount, DEST_CHAIN)
      ).to.be.revertedWith("Gateway: insufficient balance");
    });

    it("Should fail if amount is zero", async function () {
      const { gateway, user1 } = await loadFixture(deployGatewayFixture);
      const recipient = "cosmos1recipient...";

      await expect(
        gateway.connect(user1).sendToChain(recipient, 0, DEST_CHAIN)
      ).to.be.revertedWith("Gateway: amount must be greater than 0");
    });
  });

  /**
   * **Property 1: 토큰 소각 및 이벤트 발생**
   * **Requirement 1.2: Emit TransferInitiated event with sender, recipient, amount, nonce**
   */
  describe("Property 1: TransferInitiated Event", function () {
    it("Should emit TransferInitiated event with correct parameters", async function () {
      const { gateway, user1 } = await loadFixture(deployGatewayFixture);
      const amount = ethers.parseEther("100");
      const recipient = "cosmos1recipient...";

      await expect(gateway.connect(user1).sendToChain(recipient, amount, DEST_CHAIN))
        .to.emit(gateway, "TransferInitiated")
        .withArgs(
          (transferId: string) => transferId.length === 66, // bytes32 hex string
          user1.address,
          recipient,
          amount,
          1n, // nonce
          SOURCE_CHAIN,
          DEST_CHAIN,
          (blockHeight: bigint) => blockHeight > 0n,
          (timestamp: bigint) => timestamp > 0n
        );
    });

    it("Should increment nonce for each transfer", async function () {
      const { gateway, user1 } = await loadFixture(deployGatewayFixture);
      const amount = ethers.parseEther("10");
      const recipient = "cosmos1recipient...";

      expect(await gateway.getCurrentNonce()).to.equal(0);

      await gateway.connect(user1).sendToChain(recipient, amount, DEST_CHAIN);
      expect(await gateway.getCurrentNonce()).to.equal(1);

      await gateway.connect(user1).sendToChain(recipient, amount, DEST_CHAIN);
      expect(await gateway.getCurrentNonce()).to.equal(2);

      await gateway.connect(user1).sendToChain(recipient, amount, DEST_CHAIN);
      expect(await gateway.getCurrentNonce()).to.equal(3);
    });

    it("Should generate unique transfer IDs", async function () {
      const { gateway, user1 } = await loadFixture(deployGatewayFixture);
      const amount = ethers.parseEther("10");
      const recipient = "cosmos1recipient...";

      const tx1 = await gateway.connect(user1).sendToChain(recipient, amount, DEST_CHAIN);
      const receipt1 = await tx1.wait();
      const event1 = receipt1?.logs[1]; // TransferInitiated is the second event

      const tx2 = await gateway.connect(user1).sendToChain(recipient, amount, DEST_CHAIN);
      const receipt2 = await tx2.wait();
      const event2 = receipt2?.logs[1];

      // Transfer IDs should be different
      expect(event1?.topics[1]).to.not.equal(event2?.topics[1]);
    });
  });

  /**
   * **Property 1: Validation and Security**
   */
  describe("Validation and Security", function () {
    it("Should fail if recipient is empty", async function () {
      const { gateway, user1 } = await loadFixture(deployGatewayFixture);
      const amount = ethers.parseEther("100");

      await expect(
        gateway.connect(user1).sendToChain("", amount, DEST_CHAIN)
      ).to.be.revertedWith("Gateway: empty recipient");
    });

    it("Should fail if destination chain is empty", async function () {
      const { gateway, user1 } = await loadFixture(deployGatewayFixture);
      const amount = ethers.parseEther("100");
      const recipient = "cosmos1recipient...";

      await expect(
        gateway.connect(user1).sendToChain(recipient, amount, "")
      ).to.be.revertedWith("Gateway: empty destination chain");
    });

    it("Should fail if destination chain is same as source", async function () {
      const { gateway, user1 } = await loadFixture(deployGatewayFixture);
      const amount = ethers.parseEther("100");
      const recipient = "cosmos1recipient...";

      await expect(
        gateway.connect(user1).sendToChain(recipient, amount, SOURCE_CHAIN)
      ).to.be.revertedWith("Gateway: cannot transfer to same chain");
    });

    it("Should fail when paused", async function () {
      const { gateway, owner, user1 } = await loadFixture(deployGatewayFixture);
      const amount = ethers.parseEther("100");
      const recipient = "cosmos1recipient...";

      await gateway.connect(owner).pause();

      await expect(
        gateway.connect(user1).sendToChain(recipient, amount, DEST_CHAIN)
      ).to.be.revertedWith("Gateway: paused");
    });

    it("Should work after unpause", async function () {
      const { gateway, owner, user1 } = await loadFixture(deployGatewayFixture);
      const amount = ethers.parseEther("100");
      const recipient = "cosmos1recipient...";

      await gateway.connect(owner).pause();
      await gateway.connect(owner).unpause();

      await expect(
        gateway.connect(user1).sendToChain(recipient, amount, DEST_CHAIN)
      ).to.emit(gateway, "TransferInitiated");
    });
  });

  /**
   * **Property-based tests using multiple random amounts**
   */
  describe("Property-Based: Random Amount Burns", function () {
    const testAmounts = [
      ethers.parseEther("1"),
      ethers.parseEther("10"),
      ethers.parseEther("100"),
      ethers.parseEther("500"),
      ethers.parseEther("1000"),
    ];

    for (const amount of testAmounts) {
      it(`Should correctly burn ${ethers.formatEther(amount)} tokens`, async function () {
        const { token, gateway, user1 } = await loadFixture(deployGatewayFixture);
        const recipient = "cosmos1recipient...";

        const balanceBefore = await token.balanceOf(user1.address);

        await gateway.connect(user1).sendToChain(recipient, amount, DEST_CHAIN);

        const balanceAfter = await token.balanceOf(user1.address);
        expect(balanceAfter).to.equal(balanceBefore - amount);
      });
    }
  });

  /**
   * **Property-Based: Multiple sequential transfers**
   */
  describe("Property-Based: Sequential Transfers", function () {
    it("Should handle 10 sequential transfers correctly", async function () {
      const { token, gateway, user1 } = await loadFixture(deployGatewayFixture);
      const amount = ethers.parseEther("100");
      const recipient = "cosmos1recipient...";

      const initialBalance = await token.balanceOf(user1.address);
      let expectedBalance = initialBalance;

      for (let i = 0; i < 10; i++) {
        await gateway.connect(user1).sendToChain(recipient, amount, DEST_CHAIN);
        expectedBalance = expectedBalance - amount;

        expect(await token.balanceOf(user1.address)).to.equal(expectedBalance);
        expect(await gateway.getCurrentNonce()).to.equal(BigInt(i + 1));
      }
    });
  });
});
