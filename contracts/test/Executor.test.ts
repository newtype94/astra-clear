import { expect } from "chai";
import { ethers } from "hardhat";
import { loadFixture } from "@nomicfoundation/hardhat-toolbox/network-helpers";
import { BankToken, Executor } from "../typechain-types";
import { HardhatEthersSigner } from "@nomicfoundation/hardhat-ethers/signers";

/**
 * **Feature: interbank-netting-engine, Property 12: 스마트 컨트랙트 서명 검증**
 * **검증: 요구사항 5.4, 5.5**
 */
describe("Executor", function () {
  const CHAIN_ID = "bank-b";

  async function deployExecutorFixture() {
    const [owner, validator1, validator2, validator3, user1, nonValidator] = await ethers.getSigners();

    // Deploy BankToken
    const BankToken = await ethers.getContractFactory("BankToken");
    const token = await BankToken.deploy("Bank B Token", "BNKB", owner.address);
    await token.waitForDeployment();

    // Deploy Executor
    const Executor = await ethers.getContractFactory("Executor");
    const executor = await Executor.deploy(
      await token.getAddress(),
      CHAIN_ID,
      owner.address
    );
    await executor.waitForDeployment();

    // Set executor in token
    await token.setExecutor(await executor.getAddress());

    // Setup validators (3 validators, threshold = 2)
    await executor.updateValidatorSet([
      validator1.address,
      validator2.address,
      validator3.address,
    ]);

    return { token, executor, owner, validator1, validator2, validator3, user1, nonValidator };
  }

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

  describe("Deployment", function () {
    it("Should set the correct token address", async function () {
      const { token, executor } = await loadFixture(deployExecutorFixture);
      expect(await executor.token()).to.equal(await token.getAddress());
    });

    it("Should set the correct chain ID", async function () {
      const { executor } = await loadFixture(deployExecutorFixture);
      expect(await executor.chainId()).to.equal(CHAIN_ID);
    });

    it("Should initialize with correct validators", async function () {
      const { executor, validator1, validator2, validator3 } = await loadFixture(deployExecutorFixture);
      expect(await executor.getValidatorCount()).to.equal(3);
      expect(await executor.isValidator(validator1.address)).to.be.true;
      expect(await executor.isValidator(validator2.address)).to.be.true;
      expect(await executor.isValidator(validator3.address)).to.be.true;
    });

    it("Should set threshold to 2/3 of validators (rounded up)", async function () {
      const { executor } = await loadFixture(deployExecutorFixture);
      // 3 validators -> threshold = ceil(3 * 2 / 3) = 2
      expect(await executor.threshold()).to.equal(2);
    });
  });

  /**
   * **Property 12: 스마트 컨트랙트 서명 검증**
   * **Requirement 5.4: Verify all signatures using ecrecover precompile**
   */
  describe("Property 12: Signature Verification", function () {
    it("Should verify and accept valid signatures from validators", async function () {
      const { token, executor, validator1, validator2, user1 } = await loadFixture(deployExecutorFixture);

      const commandId = ethers.keccak256(ethers.toUtf8Bytes("command-1"));
      const amount = ethers.parseEther("100");

      // Get signatures from 2 validators (threshold = 2)
      const sig1 = await signMintCommand(executor, commandId, user1.address, amount, validator1);
      const sig2 = await signMintCommand(executor, commandId, user1.address, amount, validator2);

      await executor.executeMint(commandId, user1.address, amount, [sig1, sig2]);

      expect(await token.balanceOf(user1.address)).to.equal(amount);
    });

    it("Should use ecrecover to verify signatures", async function () {
      const { executor, validator1, user1 } = await loadFixture(deployExecutorFixture);

      const commandId = ethers.keccak256(ethers.toUtf8Bytes("command-2"));
      const amount = ethers.parseEther("100");

      // Manually verify the signature
      const messageHash = await executor.getMessageHash(commandId, user1.address, amount);
      const signature = await validator1.signMessage(ethers.getBytes(messageHash));

      // Recover signer from signature
      const ethSignedMessageHash = ethers.hashMessage(ethers.getBytes(messageHash));
      const recoveredSigner = ethers.recoverAddress(ethSignedMessageHash, signature);

      expect(recoveredSigner).to.equal(validator1.address);
    });
  });

  /**
   * **Property 12: 스마트 컨트랙트 서명 검증**
   * **Requirement 5.5: Reject mint command if signature verification fails**
   */
  describe("Property 12: Signature Rejection", function () {
    it("Should reject if signatures are from non-validators", async function () {
      const { executor, nonValidator, user1 } = await loadFixture(deployExecutorFixture);

      const commandId = ethers.keccak256(ethers.toUtf8Bytes("command-3"));
      const amount = ethers.parseEther("100");

      const sig1 = await signMintCommand(executor, commandId, user1.address, amount, nonValidator);
      const sig2 = await signMintCommand(executor, commandId, user1.address, amount, nonValidator);

      await expect(
        executor.executeMint(commandId, user1.address, amount, [sig1, sig2])
      ).to.be.revertedWith("Executor: insufficient valid signatures");
    });

    it("Should reject if not enough valid signatures", async function () {
      const { executor, validator1, nonValidator, user1 } = await loadFixture(deployExecutorFixture);

      const commandId = ethers.keccak256(ethers.toUtf8Bytes("command-4"));
      const amount = ethers.parseEther("100");

      // Only 1 valid signature, threshold is 2
      const sig1 = await signMintCommand(executor, commandId, user1.address, amount, validator1);
      const sig2 = await signMintCommand(executor, commandId, user1.address, amount, nonValidator);

      await expect(
        executor.executeMint(commandId, user1.address, amount, [sig1, sig2])
      ).to.be.revertedWith("Executor: insufficient valid signatures");
    });

    it("Should reject duplicate signatures from same validator", async function () {
      const { executor, validator1, user1 } = await loadFixture(deployExecutorFixture);

      const commandId = ethers.keccak256(ethers.toUtf8Bytes("command-5"));
      const amount = ethers.parseEther("100");

      // Same validator signing twice
      const sig1 = await signMintCommand(executor, commandId, user1.address, amount, validator1);

      await expect(
        executor.executeMint(commandId, user1.address, amount, [sig1, sig1])
      ).to.be.revertedWith("Executor: insufficient valid signatures");
    });

    it("Should reject if signature is for wrong command", async function () {
      const { executor, validator1, validator2, user1 } = await loadFixture(deployExecutorFixture);

      const commandId1 = ethers.keccak256(ethers.toUtf8Bytes("command-6"));
      const commandId2 = ethers.keccak256(ethers.toUtf8Bytes("command-7"));
      const amount = ethers.parseEther("100");

      // Sign for commandId1
      const sig1 = await signMintCommand(executor, commandId1, user1.address, amount, validator1);
      const sig2 = await signMintCommand(executor, commandId1, user1.address, amount, validator2);

      // Try to execute with commandId2
      await expect(
        executor.executeMint(commandId2, user1.address, amount, [sig1, sig2])
      ).to.be.revertedWith("Executor: insufficient valid signatures");
    });

    it("Should reject if signature is for wrong amount", async function () {
      const { executor, validator1, validator2, user1 } = await loadFixture(deployExecutorFixture);

      const commandId = ethers.keccak256(ethers.toUtf8Bytes("command-8"));
      const correctAmount = ethers.parseEther("100");
      const wrongAmount = ethers.parseEther("200");

      // Sign for correct amount
      const sig1 = await signMintCommand(executor, commandId, user1.address, correctAmount, validator1);
      const sig2 = await signMintCommand(executor, commandId, user1.address, correctAmount, validator2);

      // Try to execute with wrong amount
      await expect(
        executor.executeMint(commandId, user1.address, wrongAmount, [sig1, sig2])
      ).to.be.revertedWith("Executor: insufficient valid signatures");
    });

    it("Should reject if signature is for wrong recipient", async function () {
      const { executor, validator1, validator2, user1, nonValidator } = await loadFixture(deployExecutorFixture);

      const commandId = ethers.keccak256(ethers.toUtf8Bytes("command-9"));
      const amount = ethers.parseEther("100");

      // Sign for user1
      const sig1 = await signMintCommand(executor, commandId, user1.address, amount, validator1);
      const sig2 = await signMintCommand(executor, commandId, user1.address, amount, validator2);

      // Try to execute for different recipient
      await expect(
        executor.executeMint(commandId, nonValidator.address, amount, [sig1, sig2])
      ).to.be.revertedWith("Executor: insufficient valid signatures");
    });
  });

  /**
   * **Property 12: Mint Execution**
   * **Requirement 1.3: Mint exact amount to recipient**
   */
  describe("Mint Execution", function () {
    it("Should mint exact amount to recipient", async function () {
      const { token, executor, validator1, validator2, user1 } = await loadFixture(deployExecutorFixture);

      const commandId = ethers.keccak256(ethers.toUtf8Bytes("command-10"));
      const amount = ethers.parseEther("123.456");

      const sig1 = await signMintCommand(executor, commandId, user1.address, amount, validator1);
      const sig2 = await signMintCommand(executor, commandId, user1.address, amount, validator2);

      await executor.executeMint(commandId, user1.address, amount, [sig1, sig2]);

      expect(await token.balanceOf(user1.address)).to.equal(amount);
    });

    it("Should emit MintExecuted event", async function () {
      const { executor, validator1, validator2, user1 } = await loadFixture(deployExecutorFixture);

      const commandId = ethers.keccak256(ethers.toUtf8Bytes("command-11"));
      const amount = ethers.parseEther("100");

      const sig1 = await signMintCommand(executor, commandId, user1.address, amount, validator1);
      const sig2 = await signMintCommand(executor, commandId, user1.address, amount, validator2);

      await expect(executor.executeMint(commandId, user1.address, amount, [sig1, sig2]))
        .to.emit(executor, "MintExecuted")
        .withArgs(commandId, user1.address, amount, (timestamp: bigint) => timestamp > 0n);
    });

    it("Should prevent replay attacks (same command cannot be executed twice)", async function () {
      const { executor, validator1, validator2, user1 } = await loadFixture(deployExecutorFixture);

      const commandId = ethers.keccak256(ethers.toUtf8Bytes("command-12"));
      const amount = ethers.parseEther("100");

      const sig1 = await signMintCommand(executor, commandId, user1.address, amount, validator1);
      const sig2 = await signMintCommand(executor, commandId, user1.address, amount, validator2);

      // First execution
      await executor.executeMint(commandId, user1.address, amount, [sig1, sig2]);

      // Second execution should fail
      await expect(
        executor.executeMint(commandId, user1.address, amount, [sig1, sig2])
      ).to.be.revertedWith("Executor: command already processed");
    });
  });

  /**
   * **Property 13: 검증자 세트 동기화**
   * **Requirement 6.1, 6.2, 6.3, 6.4, 6.5**
   */
  describe("Property 13: Validator Set Management", function () {
    it("Should update validator set and recalculate threshold", async function () {
      const { executor, owner, validator1, validator2, validator3, user1, nonValidator } =
        await loadFixture(deployExecutorFixture);

      // Add a 4th validator
      const newValidators = [
        validator1.address,
        validator2.address,
        validator3.address,
        nonValidator.address,
      ];

      await executor.connect(owner).updateValidatorSet(newValidators);

      expect(await executor.getValidatorCount()).to.equal(4);
      // 4 validators -> threshold = ceil(4 * 2 / 3) = 3
      expect(await executor.threshold()).to.equal(3);
    });

    it("Should add a single validator", async function () {
      const { executor, owner, nonValidator } = await loadFixture(deployExecutorFixture);

      await executor.connect(owner).addValidator(nonValidator.address);

      expect(await executor.getValidatorCount()).to.equal(4);
      expect(await executor.isValidator(nonValidator.address)).to.be.true;
      // 4 validators -> threshold = 3
      expect(await executor.threshold()).to.equal(3);
    });

    it("Should remove a validator", async function () {
      const { executor, owner, validator3 } = await loadFixture(deployExecutorFixture);

      await executor.connect(owner).removeValidator(validator3.address);

      expect(await executor.getValidatorCount()).to.equal(2);
      expect(await executor.isValidator(validator3.address)).to.be.false;
      // 2 validators -> threshold = 2
      expect(await executor.threshold()).to.equal(2);
    });

    it("Should emit events on validator set changes", async function () {
      const { executor, owner, nonValidator } = await loadFixture(deployExecutorFixture);

      await expect(executor.connect(owner).addValidator(nonValidator.address))
        .to.emit(executor, "ValidatorAdded")
        .withArgs(nonValidator.address);
    });

    it("Should maintain 2/3 threshold after changes", async function () {
      const { executor, owner, nonValidator, user1 } = await loadFixture(deployExecutorFixture);

      // Start with 3 validators, threshold = 2
      expect(await executor.threshold()).to.equal(2);

      // Add 2 more validators (5 total)
      await executor.connect(owner).addValidator(nonValidator.address);
      await executor.connect(owner).addValidator(user1.address);

      // 5 validators -> threshold = ceil(5 * 2 / 3) = 4
      expect(await executor.threshold()).to.equal(4);
    });

    it("Should reject operations from removed validators", async function () {
      const { executor, owner, validator1, validator2, validator3, user1 } =
        await loadFixture(deployExecutorFixture);

      // Remove validator3
      await executor.connect(owner).removeValidator(validator3.address);

      const commandId = ethers.keccak256(ethers.toUtf8Bytes("command-13"));
      const amount = ethers.parseEther("100");

      // Try to use validator3's signature
      const sig1 = await signMintCommand(executor, commandId, user1.address, amount, validator1);
      const sig3 = await signMintCommand(executor, commandId, user1.address, amount, validator3);

      // Should fail because validator3 is no longer a validator
      await expect(
        executor.executeMint(commandId, user1.address, amount, [sig1, sig3])
      ).to.be.revertedWith("Executor: insufficient valid signatures");

      // But should work with validator2
      const sig2 = await signMintCommand(executor, commandId, user1.address, amount, validator2);
      await expect(executor.executeMint(commandId, user1.address, amount, [sig1, sig2]))
        .to.emit(executor, "MintExecuted");
    });
  });

  /**
   * **Property-Based: Threshold Calculations**
   */
  describe("Property-Based: Threshold Calculations", function () {
    const testCases = [
      { validators: 1, expectedThreshold: 1 },
      { validators: 2, expectedThreshold: 2 },
      { validators: 3, expectedThreshold: 2 },
      { validators: 4, expectedThreshold: 3 },
      { validators: 5, expectedThreshold: 4 },
      { validators: 6, expectedThreshold: 4 },
      { validators: 7, expectedThreshold: 5 },
      { validators: 10, expectedThreshold: 7 },
    ];

    for (const tc of testCases) {
      it(`${tc.validators} validators should require ${tc.expectedThreshold} signatures`, async function () {
        const signers = await ethers.getSigners();
        const [owner] = signers;

        // Deploy fresh contracts
        const BankToken = await ethers.getContractFactory("BankToken");
        const token = await BankToken.deploy("Test Token", "TEST", owner.address);
        await token.waitForDeployment();

        const Executor = await ethers.getContractFactory("Executor");
        const executor = await Executor.deploy(
          await token.getAddress(),
          "test-chain",
          owner.address
        );
        await executor.waitForDeployment();

        // Setup validators
        const validators = signers.slice(1, tc.validators + 1).map(s => s.address);
        await executor.updateValidatorSet(validators);

        expect(await executor.threshold()).to.equal(tc.expectedThreshold);
      });
    }
  });

  // =============================================================================
  // Task 12.4: Error Handling Tests
  // =============================================================================

  describe("Error Handling (Task 12.4)", function () {
    describe("validateMintCommand", function () {
      it("Should return false for already processed command", async function () {
        const { executor, validator1, validator2, user1 } = await loadFixture(deployExecutorFixture);

        const commandId = ethers.keccak256(ethers.toUtf8Bytes("test-error-cmd-1"));
        const amount = ethers.parseEther("100");

        // Create and execute first mint
        const sig1 = await signMintCommand(executor, commandId, user1.address, amount, validator1);
        const sig2 = await signMintCommand(executor, commandId, user1.address, amount, validator2);

        await executor.executeMint(commandId, user1.address, amount, [sig1, sig2]);

        // Validate should return false for already processed
        const [valid, reason] = await executor.validateMintCommand(
          commandId, user1.address, amount, [sig1, sig2]
        );
        expect(valid).to.be.false;
        expect(reason).to.equal("Command already processed");
      });

      it("Should return false for zero recipient", async function () {
        const { executor } = await loadFixture(deployExecutorFixture);

        const commandId = ethers.keccak256(ethers.toUtf8Bytes("test-error-cmd-2"));
        const amount = ethers.parseEther("100");

        const [valid, reason] = await executor.validateMintCommand(
          commandId, ethers.ZeroAddress, amount, []
        );
        expect(valid).to.be.false;
        expect(reason).to.equal("Invalid recipient address");
      });

      it("Should return false for zero amount", async function () {
        const { executor, user1 } = await loadFixture(deployExecutorFixture);

        const commandId = ethers.keccak256(ethers.toUtf8Bytes("test-error-cmd-3"));

        const [valid, reason] = await executor.validateMintCommand(
          commandId, user1.address, 0, []
        );
        expect(valid).to.be.false;
        expect(reason).to.equal("Amount must be greater than 0");
      });

      it("Should return false for insufficient signatures", async function () {
        const { executor, validator1, user1 } = await loadFixture(deployExecutorFixture);

        const commandId = ethers.keccak256(ethers.toUtf8Bytes("test-error-cmd-4"));
        const amount = ethers.parseEther("100");

        // Provide only 1 signature (threshold is 2)
        const sig1 = await signMintCommand(executor, commandId, user1.address, amount, validator1);

        const [valid, reason] = await executor.validateMintCommand(
          commandId, user1.address, amount, [sig1]
        );
        expect(valid).to.be.false;
        expect(reason).to.equal("Insufficient signatures");
      });

      it("Should return true for valid command", async function () {
        const { executor, validator1, validator2, user1 } = await loadFixture(deployExecutorFixture);

        const commandId = ethers.keccak256(ethers.toUtf8Bytes("test-error-cmd-5"));
        const amount = ethers.parseEther("100");

        const sig1 = await signMintCommand(executor, commandId, user1.address, amount, validator1);
        const sig2 = await signMintCommand(executor, commandId, user1.address, amount, validator2);

        const [valid, reason] = await executor.validateMintCommand(
          commandId, user1.address, amount, [sig1, sig2]
        );
        expect(valid).to.be.true;
        expect(reason).to.equal("");
      });
    });

    describe("getValidatorSetInfo", function () {
      it("Should return correct validator set info", async function () {
        const { executor } = await loadFixture(deployExecutorFixture);

        // Fixture already sets up 3 validators
        const [version, count, threshold] = await executor.getValidatorSetInfo();
        expect(version).to.equal(1);
        expect(count).to.equal(3);
        expect(threshold).to.equal(2);
      });
    });

    describe("verifyValidatorSet", function () {
      it("Should return false for version mismatch", async function () {
        const { executor, validator1, validator2, validator3 } = await loadFixture(deployExecutorFixture);

        const validators = [validator1.address, validator2.address, validator3.address];
        const [matches, reason] = await executor.verifyValidatorSet(validators, 0);
        expect(matches).to.be.false;
        expect(reason).to.equal("Version mismatch");
      });

      it("Should return false for validator count mismatch", async function () {
        const { executor, validator1, validator2 } = await loadFixture(deployExecutorFixture);

        const [matches, reason] = await executor.verifyValidatorSet(
          [validator1.address, validator2.address], 1
        );
        expect(matches).to.be.false;
        expect(reason).to.equal("Validator count mismatch");
      });

      it("Should return false for unknown validator", async function () {
        const { executor, validator1, validator2, nonValidator } = await loadFixture(deployExecutorFixture);

        const [matches, reason] = await executor.verifyValidatorSet(
          [validator1.address, validator2.address, nonValidator.address], 1
        );
        expect(matches).to.be.false;
        expect(reason).to.equal("Validator not found");
      });

      it("Should return true for matching validator set", async function () {
        const { executor, validator1, validator2, validator3 } = await loadFixture(deployExecutorFixture);

        const validators = [validator1.address, validator2.address, validator3.address];
        const [matches, reason] = await executor.verifyValidatorSet(validators, 1);
        expect(matches).to.be.true;
        expect(reason).to.equal("");
      });
    });

    describe("estimateMintGas", function () {
      it("Should return reasonable gas estimate", async function () {
        const { executor, user1 } = await loadFixture(deployExecutorFixture);

        const commandId = ethers.keccak256(ethers.toUtf8Bytes("test-gas"));
        const amount = ethers.parseEther("100");
        const dummySigs = ["0x" + "00".repeat(65), "0x" + "00".repeat(65)];

        const gasEstimate = await executor.estimateMintGas(
          commandId, user1.address, amount, dummySigs
        );

        // Should be at least base gas
        expect(gasEstimate).to.be.gt(50000);
        // Should include buffer
        expect(gasEstimate).to.be.lt(200000);
      });

      it("Should scale with signature count", async function () {
        const { executor, user1 } = await loadFixture(deployExecutorFixture);

        const commandId = ethers.keccak256(ethers.toUtf8Bytes("test-gas-2"));
        const amount = ethers.parseEther("100");

        const dummySig = "0x" + "00".repeat(65);
        const gas2Sigs = await executor.estimateMintGas(
          commandId, user1.address, amount, [dummySig, dummySig]
        );

        const gas5Sigs = await executor.estimateMintGas(
          commandId, user1.address, amount, [dummySig, dummySig, dummySig, dummySig, dummySig]
        );

        expect(gas5Sigs).to.be.gt(gas2Sigs);
      });
    });
  });
});
