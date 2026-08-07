package gcp

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/spf13/cobra"
)

var _ = Describe("Firewall Rules Commands", func() {
	Describe("NewVerifyFirewallRules", func() {
		var cmd interface{}

		BeforeEach(func() {
			cmd = NewVerifyFirewallRules()
		})

		It("should create a valid command", func() {
			Expect(cmd).ToNot(BeNil())
		})

		It("should have correct command metadata", func() {
			cobraCmd := cmd.(*cobra.Command)
			Expect(cobraCmd.Use).To(Equal("firewall-rules [ID]"))
			Expect(cobraCmd.Short).To(Equal("Verify firewall rules configuration"))
			Expect(cobraCmd.Long).To(ContainSubstring("Verify firewall rules configuration"))
		})
	})

	Describe("NewUpdateFirewallRules", func() {
		var cmd interface{}

		BeforeEach(func() {
			cmd = NewUpdateFirewallRules()
		})

		It("should create a valid command", func() {
			Expect(cmd).ToNot(BeNil())
		})

		It("should have correct command metadata", func() {
			cobraCmd := cmd.(*cobra.Command)
			Expect(cobraCmd.Use).To(Equal("firewall-rules [ID]"))
			Expect(cobraCmd.Short).To(Equal("Update firewall rules"))
			Expect(cobraCmd.Long).To(ContainSubstring("Update firewall rules"))
		})

		It("should have mode flag registered", func() {
			cobraCmd := cmd.(*cobra.Command)
			flag := cobraCmd.PersistentFlags().Lookup("mode")
			Expect(flag).ToNot(BeNil())
			Expect(flag.Shorthand).To(Equal("m"))
			Expect(flag.DefValue).To(Equal(ModeManual))
		})

		It("should have output-file flag registered", func() {
			cobraCmd := cmd.(*cobra.Command)
			flag := cobraCmd.PersistentFlags().Lookup("output-file")
			Expect(flag).ToNot(BeNil())
			Expect(flag.Shorthand).To(Equal("o"))
			Expect(flag.DefValue).To(BeEmpty())
		})
	})

	Describe("NewDeleteFirewallRules", func() {
		var cmd interface{}

		BeforeEach(func() {
			cmd = NewDeleteFirewallRules()
		})

		It("should create a valid command", func() {
			Expect(cmd).ToNot(BeNil())
		})

		It("should have correct command metadata", func() {
			cobraCmd := cmd.(*cobra.Command)
			Expect(cobraCmd.Use).To(Equal("firewall-rules [ID]"))
			Expect(cobraCmd.Short).To(Equal("Delete firewall rules"))
			Expect(cobraCmd.Long).To(ContainSubstring("Delete firewall rules"))
		})

		It("should have mode flag registered", func() {
			cobraCmd := cmd.(*cobra.Command)
			flag := cobraCmd.PersistentFlags().Lookup("mode")
			Expect(flag).ToNot(BeNil())
			Expect(flag.Shorthand).To(Equal("m"))
			Expect(flag.DefValue).To(Equal(ModeManual))
		})

		It("should have output-file flag registered", func() {
			cobraCmd := cmd.(*cobra.Command)
			flag := cobraCmd.PersistentFlags().Lookup("output-file")
			Expect(flag).ToNot(BeNil())
			Expect(flag.Shorthand).To(Equal("o"))
			Expect(flag.DefValue).To(BeEmpty())
		})
	})

	Describe("NewCreateFirewallRules", func() {
		var cmd interface{}

		BeforeEach(func() {
			cmd = NewCreateFirewallRules()
		})

		It("should create a valid command", func() {
			Expect(cmd).ToNot(BeNil())
		})

		It("should have correct command metadata", func() {
			cobraCmd := cmd.(*cobra.Command)
			Expect(cobraCmd.Use).To(Equal("firewall-rules"))
			Expect(cobraCmd.Short).To(Equal("Create firewall rules"))
			Expect(cobraCmd.Long).To(ContainSubstring("Create firewall rules"))
		})

		It("should have all required flags registered", func() {
			cobraCmd := cmd.(*cobra.Command)

			modeFlag := cobraCmd.PersistentFlags().Lookup("mode")
			Expect(modeFlag).ToNot(BeNil())
			Expect(modeFlag.Shorthand).To(Equal("m"))
			Expect(modeFlag.DefValue).To(Equal(ModeManual))

			outputFileFlag := cobraCmd.PersistentFlags().Lookup("output-file")
			Expect(outputFileFlag).ToNot(BeNil())
			Expect(outputFileFlag.Shorthand).To(Equal("o"))

			nameFlag := cobraCmd.PersistentFlags().Lookup("name")
			Expect(nameFlag).ToNot(BeNil())

			projectIDFlag := cobraCmd.PersistentFlags().Lookup("project-id")
			Expect(projectIDFlag).ToNot(BeNil())

			vpcNameFlag := cobraCmd.PersistentFlags().Lookup("vpc-name")
			Expect(vpcNameFlag).ToNot(BeNil())

			machineCidrFlag := cobraCmd.PersistentFlags().Lookup("machine-cidr")
			Expect(machineCidrFlag).ToNot(BeNil())

			wifConfigFlag := cobraCmd.PersistentFlags().Lookup("wif-config")
			Expect(wifConfigFlag).ToNot(BeNil())
		})
	})

	Describe("Validation functions", func() {
		BeforeEach(func() {
			// Reset args before each test
			createFwArgs = CreateFirewallRulesArgs{}
		})

		Describe("promptFirewallRuleName", func() {
			It("should return error when name is empty and not interactive", func() {
				createFwArgs.Name = ""
				createFwArgs.Interactive = false
				err := promptFirewallRuleName()
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("flag 'name' is required"))
			})

			It("should succeed when name is provided", func() {
				createFwArgs.Name = "test-firewall"
				createFwArgs.Interactive = false
				err := promptFirewallRuleName()
				Expect(err).ToNot(HaveOccurred())
			})
		})

		Describe("promptWifConfigID", func() {
			It("should return error when wif-config is empty and not interactive", func() {
				createFwArgs.WifConfigID = ""
				createFwArgs.Interactive = false
				err := promptWifConfigID()
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("flag 'wif-config' is required"))
			})

			It("should succeed when wif-config is provided", func() {
				createFwArgs.WifConfigID = "wif-123"
				createFwArgs.Interactive = false
				err := promptWifConfigID()
				Expect(err).ToNot(HaveOccurred())
			})
		})

		Describe("promptFirewallProjectID", func() {
			It("should return error when project-id is empty and not interactive", func() {
				createFwArgs.ProjectID = ""
				createFwArgs.Interactive = false
				err := promptFirewallProjectID()
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("flag 'project-id' is required"))
			})

			It("should succeed when project-id is provided", func() {
				createFwArgs.ProjectID = "test-project"
				createFwArgs.Interactive = false
				err := promptFirewallProjectID()
				Expect(err).ToNot(HaveOccurred())
			})
		})

		Describe("promptVpcName", func() {
			It("should return error when vpc-name is empty and not interactive", func() {
				createFwArgs.VpcName = ""
				createFwArgs.Interactive = false
				err := promptVpcName()
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("flag 'vpc-name' is required"))
			})

			It("should succeed when vpc-name is provided", func() {
				createFwArgs.VpcName = "test-vpc"
				createFwArgs.Interactive = false
				err := promptVpcName()
				Expect(err).ToNot(HaveOccurred())
			})
		})

		Describe("promptMachineCidr", func() {
			It("should return error when machine-cidr is empty and not interactive", func() {
				createFwArgs.MachineCidr = ""
				createFwArgs.Interactive = false
				err := promptMachineCidr()
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("flag 'machine-cidr' is required"))
			})

			It("should succeed when machine-cidr is provided", func() {
				createFwArgs.MachineCidr = "10.0.0.0/16"
				createFwArgs.Interactive = false
				err := promptMachineCidr()
				Expect(err).ToNot(HaveOccurred())
			})
		})
	})
})
