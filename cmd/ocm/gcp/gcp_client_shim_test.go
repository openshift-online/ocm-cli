package gcp

import (
	"fmt"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	cloudresourcemanager "google.golang.org/api/cloudresourcemanager/v1"
)

var _ = Describe("GcpClientShim helper unit tests", func() {
	Describe("removeMemberFromBinding", func() {
		const (
			testMember = "b"
		)
		var (
			testSubject *shim
			testBinding *cloudresourcemanager.Binding
		)
		BeforeEach(func() {
			testSubject = &shim{}
			testBinding = &cloudresourcemanager.Binding{
				Members: []string{"a", testMember, "c"},
			}
		})
		It("does not modify the member list of the binding parameter when member is absent", func() {
			beforeLen := len(testBinding.Members)
			modified := testSubject.removeMemberFromBinding(testMember, testBinding)
			Expect(modified).To(BeTrue(), "the method should report the modification")
			Expect(len(testBinding.Members)).To(Equal(beforeLen-1),
				"there should be one less item in the list")
			for _, member := range testBinding.Members {
				if member == testMember {
					Fail(fmt.Sprintf("removed member should no longer be present in list: %v", testBinding.Members))
				}
			}
		})
		It("modifies the member list of the passed binding when member is present", func() {
			beforeLen := len(testBinding.Members)
			modified := testSubject.removeMemberFromBinding("foobar", testBinding)
			Expect(modified).To(BeFalse(), "the method should report that no modification occurred")
			Expect(len(testBinding.Members)).To(Equal(beforeLen),
				"there should be the same number of items in the list")
		})
	})
	Describe("applyMemberToRoleInPolicy", func() {
		const (
			testMember = "b"
			testRole   = "testRole"
		)
		var (
			testSubject    *shim
			testMembers    []string
			funcCalled     bool
			funcParameters struct {
				member  string
				binding *cloudresourcemanager.Binding
			}
		)
		BeforeEach(func() {
			testSubject = &shim{}
			testMembers = []string{"a", testMember, "c"}
			funcCalled = false
		})
		It("calls the apply function if the role and member is present", func() {
			modified := testSubject.applyMemberToRoleInPolicy(&cloudresourcemanager.Policy{
				Bindings: []*cloudresourcemanager.Binding{
					{
						Role:    testRole,
						Members: testMembers,
					},
				},
			}, testRole, testMember, func(member string, binding *cloudresourcemanager.Binding) bool {
				funcCalled = true
				funcParameters.binding = binding
				funcParameters.member = member
				return true
			})
			Expect(modified).To(BeTrue(), "call should have returned what the apply function returned.")
			Expect(funcCalled).To(BeTrue(), "apply function should have been called")
			Expect(funcParameters.binding.Role).To(Equal(testRole),
				"apply function should have been called with the expected binding")
			Expect(funcParameters.member).To(Equal(testMember),
				"apply function should have been called with the expected binding")
		})
		It("does not call the apply function if the role is not present", func() {
			modified := testSubject.applyMemberToRoleInPolicy(&cloudresourcemanager.Policy{
				Bindings: []*cloudresourcemanager.Binding{
					{
						Role:    "foobar",
						Members: testMembers,
					},
				},
			}, testRole, testMember, func(member string, binding *cloudresourcemanager.Binding) bool {
				funcCalled = true
				funcParameters.binding = binding
				funcParameters.member = member
				return true
			})
			Expect(funcCalled).To(BeFalse(), "apply function should not have been called")
			Expect(modified).To(BeFalse(), "no modification should have occurred")
		})
		It("does not call the apply function if the member is not present in the role", func() {
			modified := testSubject.applyMemberToRoleInPolicy(&cloudresourcemanager.Policy{
				Bindings: []*cloudresourcemanager.Binding{
					{
						Role:    testRole,
						Members: []string{},
					},
				},
			}, testRole, testMember, func(member string, binding *cloudresourcemanager.Binding) bool {
				funcCalled = true
				funcParameters.binding = binding
				funcParameters.member = member
				for _, presentMember := range binding.Members {
					if presentMember == member {
						return true
					}
				}
				return false
			})
			Expect(funcCalled).To(BeTrue(), "the apply function would have been called")
			Expect(modified).To(BeFalse(), "no modification should have occurred")
		})
	})
})
