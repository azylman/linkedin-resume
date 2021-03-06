package main

import (
	"errors"
	"fmt"
	"strings"

	"github.com/PuerkitoBio/goquery"
	goresume "github.com/azylman/gopher-resume"
)

func resumeForURL(url string) (goresume.Resume, error) {
	var resume goresume.Resume

	doc, err := goquery.NewDocument(url)
	if err != nil {
		return resume, err
	}

	basic, err := extractBasicInfo(doc)
	if err != nil {
		return resume, err
	}

	works, err := extractWorks(doc)
	if err != nil {
		return resume, err
	}

	educations, err := extractEducations(doc)
	if err != nil {
		return resume, err
	}

	skills, err := extractSkills(doc)
	if err != nil {
		return resume, err
	}

	languages, err := extractLanguages(doc)
	if err != nil {
		return resume, err
	}

	interests, err := extractInterests(doc)
	if err != nil {
		return resume, err
	}

	resume.BasicInformation = basic
	resume.WorkExperience = works
	resume.EducationHistory = educations
	resume.Skills = skills
	resume.Languages = languages
	resume.Interests = interests

	resume.BasicInformation.Profiles = append(resume.BasicInformation.Profiles, goresume.Profile{
		Network: "LinkedIn",
		URL:     url,
	})

	// Some themes require these
	resume.VolunteerExperience = []goresume.Volunteer{}
	resume.Awards = []goresume.Award{}
	resume.Publications = []goresume.Publication{}

	return resume, nil
}

func extractBasicInfo(doc *goquery.Document) (goresume.Basic, error) {
	var basic goresume.Basic

	name := findOne(doc.Selection, "#name")
	if name == nil {
		return basic, errors.New("unable to find name")
	}
	basic.Name = strings.TrimSpace(name.Text())

	label := findOne(doc.Selection, ".headline.title")
	if label == nil {
		return basic, errors.New("unable to find label")
	}
	basic.Label = strings.TrimSpace(label.Text())

	picture := findOne(doc.Selection, ".profile-picture img")
	if picture == nil {
		return basic, errors.New("unable to find picture")
	}

	picUrl, ok := picture.Attr("data-delayed-url")
	if !ok {
		return basic, errors.New("unable to find picture location")
	}
	basic.Picture = picUrl

	summary := findOne(doc.Selection, "#summary .description p")
	if summary == nil {
		return basic, errors.New("unable to find summary")
	}

	h, err := summary.Html()
	if err != nil {
		return basic, err
	}

	h = strings.Replace(h, "<br/>", "\n", -1)

	basic.Summary = strings.TrimSpace(h)

	region := findOne(doc.Selection, "#demographics .locality")
	if region == nil {
		return basic, errors.New("unable to find region")
	}

	basic.Location.Region = strings.TrimSpace(region.Text())
	basic.Profiles = []goresume.Profile{} // Some themes break if this isn't set

	return basic, nil
}

func extractWorks(doc *goquery.Document) ([]goresume.Work, error) {
	works := []goresume.Work{}
	var err error

	doc.Find("#experience .position").Each(func(i int, s *goquery.Selection) {
		work, e := extractWork(s)
		if e != nil {
			err = e
			return
		}
		works = append(works, work)
	})

	return works, err
}

func extractWork(s *goquery.Selection) (goresume.Work, error) {
	var work goresume.Work

	company := findOne(s, "header .item-subtitle span")
	if company == nil {
		return work, errors.New("unable to find company")
	}
	work.Company = strings.TrimSpace(company.Text())

	position := findOne(s, "header .item-title span")
	if position == nil {
		return work, errors.New("unable to find position")
	}
	work.Position = strings.TrimSpace(position.Text())

	dateRange := findOne(s, ".meta .date-range")
	if dateRange == nil {
		return work, errors.New("unable to find date range")
	}

	pieces := strings.Split(dateRange.Text(), "–")
	if len(pieces) != 2 {
		return work, fmt.Errorf("invalid date range '%s': %#v", dateRange.Text(), pieces)
	}
	work.StartDate = strings.TrimSpace(pieces[0])
	ends := strings.Split(pieces[1], "(") // Remove the length of time it's been a job
	work.EndDate = strings.TrimSpace(ends[0])

	summary := findOne(s, ".description")
	if summary == nil {
		return work, errors.New("unable to find summary")
	}

	work.Summary = strings.TrimSpace(summary.Text())
	work.Highlights = []string{} // Some themes require this to exist

	return work, nil
}

func extractEducations(doc *goquery.Document) ([]goresume.Education, error) {
	educations := []goresume.Education{}
	var err error

	doc.Find("#education .school").Each(func(i int, s *goquery.Selection) {
		education, e := extractEducation(s)
		if e != nil {
			err = e
			return
		}
		educations = append(educations, education)
	})

	return educations, err
}

func extractEducation(s *goquery.Selection) (goresume.Education, error) {
	var education goresume.Education

	institution := findOne(s, "header .item-title span")
	if institution == nil {
		return education, errors.New("unable to find institution")
	}

	education.Institution = strings.TrimSpace(institution.Text())

	dateRange := findOne(s, ".meta .date-range")
	if dateRange == nil {
		return education, errors.New("unable to find date range")
	}

	pieces := strings.Split(dateRange.Text(), "–")
	if len(pieces) != 2 {
		return education, fmt.Errorf("invalid date range '%s': %#v", dateRange.Text(), pieces)
	}
	education.StartDate = strings.TrimSpace(pieces[0])
	education.EndDate = strings.TrimSpace(pieces[1])

	degree := findOne(s, "header .item-subtitle span")
	if degree == nil {
		return education, errors.New("unable to find degree")
	}
	// Degree is a free-form field and can be formatted any way. For now, assume it's in the format
	// StudyType, Area - e.g. Bachelor's Degree, Computer Engineering
	pieces = strings.Split(degree.Text(), ", ")
	education.StudyType = strings.TrimSpace(pieces[0])
	education.Area = strings.TrimSpace(pieces[1])

	return education, nil
}

func extractSkills(doc *goquery.Document) ([]goresume.Skill, error) {
	skills := []goresume.Skill{}
	doc.Find("#skills .skill").Each(func(i int, s *goquery.Selection) {
		skills = append(skills, goresume.Skill{Name: s.Text()})
	})
	return skills, nil
}

func extractLanguages(doc *goquery.Document) ([]goresume.Language, error) {
	languages := []goresume.Language{}
	var err error

	doc.Find("#languages .language").Each(func(i int, s *goquery.Selection) {
		language, e := extractLanguage(s)
		if e != nil {
			err = e
			return
		}
		languages = append(languages, language)
	})

	return languages, err
}

func extractLanguage(s *goquery.Selection) (goresume.Language, error) {
	var language goresume.Language

	name := findOne(s, ".name")
	if name == nil {
		return language, errors.New("unable to find language name")
	}
	language.Name = strings.TrimSpace(name.Text())

	proficiency := findOne(s, ".proficiency")
	if proficiency == nil {
		return language, errors.New("unable to find language proficiency")
	}
	language.Level = strings.TrimSpace(proficiency.Text())

	return language, nil
}

func extractInterests(doc *goquery.Document) ([]goresume.Interest, error) {
	interests := []goresume.Interest{}
	doc.Find("#interests .interest").Each(func(i int, s *goquery.Selection) {
		interests = append(interests, goresume.Interest{Name: s.Text()})
	})
	return interests, nil
}
