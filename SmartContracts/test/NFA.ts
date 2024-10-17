import {
  time,
  loadFixture,
} from "@nomicfoundation/hardhat-toolbox/network-helpers";
import { anyValue } from "@nomicfoundation/hardhat-chai-matchers/withArgs";
import { expect } from "chai";
import hre from "hardhat";
import { createTestNFA } from "../lib/createTestContract";

let nfaAddress;

describe("NFA", function () {
  console.log("NFA_FACTORY_CONTRACT_ADDRESS: ", process.env.NFA_FACTORY_CONTRACT_ADDRESS)
  // We define a fixture to reuse the same setup in every test.
  // We use loadFixture to run this setup once, snapshot that state,
  // and reset Hardhat Network to that snapshot in every test.
  async function deployNFAFixture() {

    const [deployer] = await hre.ethers.getSigners();

    const NFAFactoryFactory = await hre.ethers.getContractFactory("NFAFactory");

    const nfaFactoryFactory = await NFAFactoryFactory.attach(
      process.env.NFA_FACTORY_CONTRACT_ADDRESS
    );

    nfaAddress = await createTestNFA(deployer);

    const NFAFactory = await hre.ethers.getContractFactory("NFA");

    const testNFA = await NFAFactory.attach(nfaAddress);

    return { nfaAddress, testNFA, nfaFactoryFactory };
  }

  this.beforeAll(async () => {
    await deployNFAFixture();
  });

  describe("Deployment", function () {
    it("Should set the right code hash", async function () {
      
      const { nfaAddress, testNFA, nfaFactoryFactory: nfaFactory} = await deployNFAFixture();


      const appInfo = await testNFA.getAppInfo(nfaAddress);
      await appInfo.wait();

      console.log("appInfo: ", appInfo);
      const codeHash =  appInfo.codeHash;

      expect(codeHash).to.equal(
        "0x518c4bf773cea6b73b940ff8525167d33343aecfef4edb56d928fd77aa6d89f1"
      );
    });
  });
});