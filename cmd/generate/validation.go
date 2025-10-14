package main

import (
	"fmt"
	"slices"

	"github.com/urfave/cli/v2"
)

type flagValidation func(ctx *cli.Context) error
type flagValidations []flagValidation

func newFlagValidations() flagValidations {
	return flagValidations{}
}

func (v flagValidations) atLeastOne(flags ...string) flagValidations {
	validateFunc := func(ctx *cli.Context) error {
		if slices.ContainsFunc(flags, ctx.IsSet) {
			return nil
		}
		return fmt.Errorf("at least one of flags %s must be specified", flags)
	}
	return append(v, validateFunc)
}

// requiredWhenSet checks that the second flag is set if the first flag is set
func (v flagValidations) requiredWhenSet(setFlag, shouldAlsoBeSetFlag string) flagValidations {
	validateFunc := func(ctx *cli.Context) error {
		if ctx.IsSet(setFlag) && !ctx.IsSet(shouldAlsoBeSetFlag) {
			return fmt.Errorf("flag %s is required when flag %s is set", shouldAlsoBeSetFlag, setFlag)
		}
		return nil
	}
	return append(v, validateFunc)
}

func (v flagValidations) conflicting(firstGroup []string, secondGroup []string) flagValidations {
	validateFunc := func(ctx *cli.Context) error {
		var isSetFirstGroup []string
		for _, flag := range firstGroup {
			if ctx.IsSet(flag) {
				isSetFirstGroup = append(isSetFirstGroup, flag)
			}
		}
		for _, flag := range secondGroup {
			if ctx.IsSet(flag) {
				if len(isSetFirstGroup) > 0 {
					return fmt.Errorf("flags %v and %v are mutually exclusive", firstGroup, secondGroup)
				}
			}
		}
		return nil
	}
	return append(v, validateFunc)
}

func (v flagValidations) validate(ctx *cli.Context) error {
	for _, f := range v {
		if err := f(ctx); err != nil {
			return err
		}
	}
	return nil
}
