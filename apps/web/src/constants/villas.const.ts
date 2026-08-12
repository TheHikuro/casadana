import type { IconName } from "lucide-react/dynamic"

import BATHROOM from "@/assets/casadana/bathroom.jpeg"
import BEDROOM_1 from "@/assets/casadana/bedroom1.jpeg"
import BEDROOM_2 from "@/assets/casadana/bedroom1_1.jpeg"
import BEDROOM_3 from "@/assets/casadana/bedroom1_2.jpg"
import BEDROOM_4 from "@/assets/casadana/bedroom1_3.jpg"
import BEDROOM_5 from "@/assets/casadana/bedroom2.jpeg"
import BEDROOM_6 from "@/assets/casadana/bedroom3.jpeg"
import FRONT_1 from "@/assets/casadana/front1.jpeg"
import FRONT_2 from "@/assets/casadana/front2.jpeg"
import FRONT_3 from "@/assets/casadana/front3.jpeg"
import JACUZZI_1 from "@/assets/casadana/jacuzzi1.jpeg"
import JACUZZI_2 from "@/assets/casadana/jacuzzi2.jpeg"
import KITCHEN_1 from "@/assets/casadana/kitchen1.jpeg"
import KITCHEN_2 from "@/assets/casadana/kitchen2.jpeg"
import LIVING_1 from "@/assets/casadana/living_room1.jpeg"
import LIVING_2 from "@/assets/casadana/living_room2.jpeg"
import LIVING_3 from "@/assets/casadana/living_room3.jpeg"
import LIVING_4 from "@/assets/casadana/living_room4.jpg"
import ROOFTOP_1 from "@/assets/casadana/rooftop1.jpeg"
import ROOFTOP_3 from "@/assets/casadana/rooftop3.jpeg"
import ROOFTOP_5 from "@/assets/casadana/rooftop5.jpeg"
import ROOFTOP_7 from "@/assets/casadana/rooftop7.jpeg"
import POOL_1 from "@/assets/casadessy/pool1.jpeg"
import { m } from "@/paraglide/messages"

import { GalleryCategory } from "./gallery-categories.const"

export interface GalleryEntry {
  src: string
  label: string
  category: GalleryCategory
}

export interface BentoTile {
  category: GalleryCategory
  src: string
  index: string
  label: string
  caption: string
  span: "wide" | "tall" | "half" | "third" | "quarter" | "wideFlat"
  placeholder?: boolean
  placeholderLabel?: string
}

export interface PointOfInterest {
  number: string
  title: string
  description: string
  distance: string
}

export interface ExperienceEntry {
  number: string
  tag: string
  title: string
  description: string
  location: string
  icon: "kart" | "wind" | "splash" | "kayak" | "fairway" | "olive" | "salt" | "spa"
}

export interface ReviewEntry {
  initial: string
  name: string
  meta: string
  quote: string
  source: string
  stars: number
}

export interface FaqEntry {
  question: string
  answer: string
}

export interface VillaFeature {
  icon: IconName
  label: string
}

export interface VillaStat {
  label: string
  value: string
}

export interface VillaData {
  id: "casadana" | "casacasay"
  index: string
  badge: string
  brandSub: string
  hero: {
    image: string
    eyebrow: Array<string>
    titlePrefix: string
    titleItalic: string
    titleSuffix: string
    stats: Array<VillaStat>
    price: number
    priceLabel: string
  }
  ribbon: Array<string>
  about: {
    chapter: string
    titleLead: string
    titleItalic: string
    lead: string
    /** Body copy, one entry per rendered paragraph. */
    body: Array<string>
    features: Array<VillaFeature>
    meta: Array<VillaStat>
  }
  booking: {
    nightly: number
    rating: number
    reviewCount: number
    maxGuests: number
    defaultGuests: number
  }
  gallery: {
    chapter: string
    titleItalic: string
    titleTail: string
    description: string
    totalLabel: string
    tiles: Array<BentoTile>
    images: Record<GalleryCategory, Array<GalleryEntry>>
  }
  localArea: {
    chapter: string
    titleLead: string
    titleItalic: string
    description: string
    mainImage: string
    overlapImage?: string
    stampBig: string
    stampSmall: string
    points: Array<PointOfInterest>
  }
  experiences?: {
    chapter: string
    titleLead: string
    titleItalic: string
    description: string
    entries: Array<ExperienceEntry>
  }
  reviews: {
    chapter: string
    titleLead: string
    titleItalic: string
    description: string
    score: number
    count: number
    bars: Array<{ label: string; pct: number; value: string }>
    entries: Array<ReviewEntry>
  }
  faq?: {
    chapter: string
    titleLead: string
    titleItalic: string
    intro: string
    entries: Array<FaqEntry>
  }
  ctaStrip: {
    lead: string
    italic: string
    tail: string
    button: string
  }
  sister: {
    chapter: string
    titleLead: string
    titleItalic: string
    description: string
    button: string
    targetId: "casadana" | "casacasay"
    image: string
  }
}

function buildCasadana(): VillaData {
  return {
    id: "casadana",
    index: m.villa_casadana_index(),
    badge: m.villa_casadana_badge(),
    brandSub: m.villa_casadana_brand_sub(),
    hero: {
      image: ROOFTOP_7,
      eyebrow: [
        m.villa_casadana_hero_eyebrow_1(),
        m.villa_casadana_hero_eyebrow_2(),
        m.villa_casadana_hero_eyebrow_3(),
      ],
      titlePrefix: m.villa_casadana_hero_title_prefix(),
      titleItalic: m.villa_casadana_hero_title_italic(),
      titleSuffix: m.villa_casadana_hero_title_suffix(),
      stats: [
        {
          label: m.villa_casadana_hero_stat_1_label(),
          value: m.villa_casadana_hero_stat_1_value(),
        },
        {
          label: m.villa_casadana_hero_stat_2_label(),
          value: m.villa_casadana_hero_stat_2_value(),
        },
        {
          label: m.villa_casadana_hero_stat_3_label(),
          value: m.villa_casadana_hero_stat_3_value(),
        },
        {
          label: m.villa_casadana_hero_stat_4_label(),
          value: m.villa_casadana_hero_stat_4_value(),
        },
        {
          label: m.villa_casadana_hero_stat_5_label(),
          value: m.villa_casadana_hero_stat_5_value(),
        },
      ],
      price: 185,
      priceLabel: m.villa_casadana_hero_price_label(),
    },
    ribbon: [
      m.villa_casadana_ribbon_1(),
      m.villa_casadana_ribbon_2(),
      m.villa_casadana_ribbon_3(),
      m.villa_casadana_ribbon_4(),
      m.villa_casadana_ribbon_5(),
      m.villa_casadana_ribbon_6(),
    ],
    about: {
      chapter: m.villa_casadana_about_chapter(),
      titleLead: m.villa_casadana_about_title_lead(),
      titleItalic: m.villa_casadana_about_title_italic(),
      lead: m.villa_casadana_about_lead(),
      body: [
        m.villa_casadana_about_body(),
        m.villa_casadana_about_body_2(),
        m.villa_casadana_about_body_3(),
      ],
      features: [
        { icon: "bath", label: m.villa_casadana_about_feature_jacuzzi() },
        { icon: "sun", label: m.villa_casadana_about_feature_solarium() },
        { icon: "snowflake", label: m.villa_casadana_about_feature_ac() },
        { icon: "wifi", label: m.villa_casadana_about_feature_wifi() },
        { icon: "flame", label: m.villa_casadana_about_feature_bbq() },
        { icon: "car", label: m.villa_casadana_about_feature_parking() },
        { icon: "waves", label: m.villa_casadana_about_feature_beach() },
        { icon: "bed", label: m.villa_casadana_about_feature_linens() },
      ],
      meta: [
        {
          label: m.villa_casadana_about_meta_1_label(),
          value: m.villa_casadana_about_meta_1_value(),
        },
        {
          label: m.villa_casadana_about_meta_2_label(),
          value: m.villa_casadana_about_meta_2_value(),
        },
        {
          label: m.villa_casadana_about_meta_3_label(),
          value: m.villa_casadana_about_meta_3_value(),
        },
      ],
    },
    booking: {
      nightly: 95,
      rating: 4.96,
      reviewCount: 87,
      maxGuests: 6,
      defaultGuests: 4,
    },
    gallery: {
      chapter: m.villa_casadana_gallery_chapter(),
      titleItalic: m.villa_casadana_gallery_title_italic(),
      titleTail: m.villa_casadana_gallery_title_tail(),
      description: m.villa_casadana_gallery_description(),
      totalLabel: m.villa_casadana_gallery_total_label(),
      tiles: [
        {
          category: "LIVING_SPACES",
          src: LIVING_1,
          index: m.villa_casadana_gallery_tile_1_index(),
          label: m.villa_casadana_gallery_tile_1_index(),
          caption: m.villa_casadana_gallery_tile_1_caption(),
          span: "wide",
        },
        {
          category: "BEDROOMS",
          src: BEDROOM_3,
          index: m.villa_casadana_gallery_tile_2_index(),
          label: m.villa_casadana_gallery_tile_2_index(),
          caption: m.villa_casadana_gallery_tile_2_caption(),
          span: "half",
        },
        {
          category: "KITCHEN",
          src: KITCHEN_1,
          index: m.villa_casadana_gallery_tile_3_index(),
          label: m.villa_casadana_gallery_tile_3_index(),
          caption: m.villa_casadana_gallery_tile_3_caption(),
          span: "half",
        },
        {
          category: "BATHROOMS",
          src: BATHROOM,
          index: m.villa_casadana_gallery_tile_4_index(),
          label: m.villa_casadana_gallery_tile_4_index(),
          caption: m.villa_casadana_gallery_tile_4_caption(),
          span: "third",
        },
        {
          category: "OUTDOOR",
          src: ROOFTOP_1,
          index: m.villa_casadana_gallery_tile_5_index(),
          label: m.villa_casadana_gallery_tile_5_index(),
          caption: m.villa_casadana_gallery_tile_5_caption(),
          span: "wideFlat",
        },
      ],
      images: {
        LIVING_SPACES: [
          {
            src: LIVING_1,
            label: m.villa_casadana_gallery_img_living_1(),
            category: "LIVING_SPACES",
          },
          {
            src: LIVING_2,
            label: m.villa_casadana_gallery_img_living_2(),
            category: "LIVING_SPACES",
          },
          {
            src: LIVING_4,
            label: m.villa_casadana_gallery_img_living_3(),
            category: "LIVING_SPACES",
          },
          {
            src: LIVING_3,
            label: m.villa_casadana_gallery_img_living_4(),
            category: "LIVING_SPACES",
          },
        ],
        BEDROOMS: [
          { src: BEDROOM_3, label: m.villa_casadana_gallery_img_bedroom_1(), category: "BEDROOMS" },
          { src: BEDROOM_2, label: m.villa_casadana_gallery_img_bedroom_2(), category: "BEDROOMS" },
          { src: BEDROOM_5, label: m.villa_casadana_gallery_img_bedroom_3(), category: "BEDROOMS" },
          { src: BEDROOM_6, label: m.villa_casadana_gallery_img_bedroom_4(), category: "BEDROOMS" },
          { src: BEDROOM_1, label: m.villa_casadana_gallery_img_bedroom_5(), category: "BEDROOMS" },
          { src: BEDROOM_4, label: m.villa_casadana_gallery_img_bedroom_6(), category: "BEDROOMS" },
        ],
        KITCHEN: [
          { src: KITCHEN_1, label: m.villa_casadana_gallery_img_kitchen_1(), category: "KITCHEN" },
          { src: KITCHEN_2, label: m.villa_casadana_gallery_img_kitchen_2(), category: "KITCHEN" },
        ],
        BATHROOMS: [
          {
            src: BATHROOM,
            label: m.villa_casadana_gallery_img_bathroom_1(),
            category: "BATHROOMS",
          },
        ],
        OUTDOOR: [
          { src: ROOFTOP_7, label: m.villa_casadana_gallery_img_outdoor_1(), category: "OUTDOOR" },
          { src: ROOFTOP_1, label: m.villa_casadana_gallery_img_outdoor_2(), category: "OUTDOOR" },
          { src: ROOFTOP_3, label: m.villa_casadana_gallery_img_outdoor_3(), category: "OUTDOOR" },
          { src: ROOFTOP_5, label: m.villa_casadana_gallery_img_outdoor_4(), category: "OUTDOOR" },
          { src: JACUZZI_1, label: m.villa_casadana_gallery_img_outdoor_5(), category: "OUTDOOR" },
          { src: JACUZZI_2, label: m.villa_casadana_gallery_img_outdoor_6(), category: "OUTDOOR" },
          { src: FRONT_2, label: m.villa_casadana_gallery_img_outdoor_7(), category: "OUTDOOR" },
          { src: FRONT_1, label: m.villa_casadana_gallery_img_outdoor_8(), category: "OUTDOOR" },
          { src: FRONT_3, label: m.villa_casadana_gallery_img_outdoor_9(), category: "OUTDOOR" },
        ],
        DINING: [],
        UTILITY: [],
      },
    },
    localArea: {
      chapter: m.villa_casadana_local_area_chapter(),
      titleLead: m.villa_casadana_local_area_title_lead(),
      titleItalic: m.villa_casadana_local_area_title_italic(),
      description: m.villa_casadana_local_area_description(),
      mainImage: FRONT_2,
      overlapImage: JACUZZI_1,
      stampBig: m.villa_casadana_local_area_stamp_big(),
      stampSmall: m.villa_casadana_local_area_stamp_small(),
      points: [
        {
          number: "№ 01",
          title: m.villa_casadana_local_area_point_1_title(),
          description: m.villa_casadana_local_area_point_1_description(),
          distance: m.villa_casadana_local_area_point_1_distance(),
        },
        {
          number: "№ 02",
          title: m.villa_casadana_local_area_point_2_title(),
          description: m.villa_casadana_local_area_point_2_description(),
          distance: m.villa_casadana_local_area_point_2_distance(),
        },
        {
          number: "№ 03",
          title: m.villa_casadana_local_area_point_3_title(),
          description: m.villa_casadana_local_area_point_3_description(),
          distance: m.villa_casadana_local_area_point_3_distance(),
        },
        {
          number: "№ 04",
          title: m.villa_casadana_local_area_point_4_title(),
          description: m.villa_casadana_local_area_point_4_description(),
          distance: m.villa_casadana_local_area_point_4_distance(),
        },
        {
          number: "№ 05",
          title: m.villa_casadana_local_area_point_5_title(),
          description: m.villa_casadana_local_area_point_5_description(),
          distance: m.villa_casadana_local_area_point_5_distance(),
        },
      ],
    },
    experiences: {
      chapter: m.villa_casadana_experiences_chapter(),
      titleLead: m.villa_casadana_experiences_title_lead(),
      titleItalic: m.villa_casadana_experiences_title_italic(),
      description: m.villa_casadana_experiences_description(),
      entries: [
        {
          number: "№ 01",
          tag: m.villa_casadana_experiences_1_tag(),
          title: m.villa_casadana_experiences_1_title(),
          description: m.villa_casadana_experiences_1_description(),
          location: m.villa_casadana_experiences_1_location(),
          icon: "kart",
        },
        {
          number: "№ 02",
          tag: m.villa_casadana_experiences_2_tag(),
          title: m.villa_casadana_experiences_2_title(),
          description: m.villa_casadana_experiences_2_description(),
          location: m.villa_casadana_experiences_2_location(),
          icon: "wind",
        },
        {
          number: "№ 03",
          tag: m.villa_casadana_experiences_3_tag(),
          title: m.villa_casadana_experiences_3_title(),
          description: m.villa_casadana_experiences_3_description(),
          location: m.villa_casadana_experiences_3_location(),
          icon: "splash",
        },
        {
          number: "№ 04",
          tag: m.villa_casadana_experiences_4_tag(),
          title: m.villa_casadana_experiences_4_title(),
          description: m.villa_casadana_experiences_4_description(),
          location: m.villa_casadana_experiences_4_location(),
          icon: "kayak",
        },
      ],
    },
    reviews: {
      chapter: m.villa_casadana_reviews_chapter(),
      titleLead: m.villa_casadana_reviews_title_lead(),
      titleItalic: m.villa_casadana_reviews_title_italic(),
      description: m.villa_casadana_reviews_description(),
      score: 4.96,
      count: 87,
      bars: [
        { label: m.villa_casadana_reviews_bar_1_label(), pct: 99, value: "5.0" },
        { label: m.villa_casadana_reviews_bar_2_label(), pct: 98, value: "4.9" },
        { label: m.villa_casadana_reviews_bar_3_label(), pct: 96, value: "4.9" },
        { label: m.villa_casadana_reviews_bar_4_label(), pct: 100, value: "5.0" },
        { label: m.villa_casadana_reviews_bar_5_label(), pct: 94, value: "4.8" },
      ],
      entries: [
        {
          initial: "C",
          name: m.villa_casadana_reviews_1_name(),
          meta: m.villa_casadana_reviews_1_meta(),
          quote: m.villa_casadana_reviews_1_quote(),
          source: m.villa_casadana_reviews_1_source(),
          stars: 5,
        },
        {
          initial: "M",
          name: m.villa_casadana_reviews_2_name(),
          meta: m.villa_casadana_reviews_2_meta(),
          quote: m.villa_casadana_reviews_2_quote(),
          source: m.villa_casadana_reviews_2_source(),
          stars: 5,
        },
        {
          initial: "J",
          name: m.villa_casadana_reviews_3_name(),
          meta: m.villa_casadana_reviews_3_meta(),
          quote: m.villa_casadana_reviews_3_quote(),
          source: m.villa_casadana_reviews_3_source(),
          stars: 5,
        },
      ],
    },
    faq: {
      chapter: m.villa_casadana_faq_chapter(),
      titleLead: m.villa_casadana_faq_title_lead(),
      titleItalic: m.villa_casadana_faq_title_italic(),
      intro: m.villa_casadana_faq_intro(),
      entries: [
        { question: m.villa_casadana_faq_1_question(), answer: m.villa_casadana_faq_1_answer() },
        { question: m.villa_casadana_faq_2_question(), answer: m.villa_casadana_faq_2_answer() },
        { question: m.villa_casadana_faq_3_question(), answer: m.villa_casadana_faq_3_answer() },
        { question: m.villa_casadana_faq_4_question(), answer: m.villa_casadana_faq_4_answer() },
        { question: m.villa_casadana_faq_5_question(), answer: m.villa_casadana_faq_5_answer() },
        { question: m.villa_casadana_faq_6_question(), answer: m.villa_casadana_faq_6_answer() },
      ],
    },
    ctaStrip: {
      lead: m.villa_casadana_cta_strip_lead(),
      italic: m.villa_casadana_cta_strip_italic(),
      tail: m.villa_casadana_cta_strip_tail(),
      button: m.villa_casadana_cta_strip_button(),
    },
    sister: {
      chapter: m.villa_casadana_sister_chapter(),
      titleLead: m.villa_casadana_sister_title_lead(),
      titleItalic: m.villa_casadana_sister_title_italic(),
      description: m.villa_casadana_sister_description(),
      button: m.villa_casadana_sister_button(),
      targetId: "casacasay",
      image: POOL_1,
    },
  }
}

function buildCasacasay(): VillaData {
  return {
    id: "casacasay",
    index: m.villa_casacasay_index(),
    badge: m.villa_casacasay_badge(),
    brandSub: m.villa_casacasay_brand_sub(),
    hero: {
      image: POOL_1,
      eyebrow: [
        m.villa_casacasay_hero_eyebrow_1(),
        m.villa_casacasay_hero_eyebrow_2(),
        m.villa_casacasay_hero_eyebrow_3(),
      ],
      titlePrefix: m.villa_casacasay_hero_title_prefix(),
      titleItalic: m.villa_casacasay_hero_title_italic(),
      titleSuffix: m.villa_casacasay_hero_title_suffix(),
      stats: [
        {
          label: m.villa_casacasay_hero_stat_1_label(),
          value: m.villa_casacasay_hero_stat_1_value(),
        },
        {
          label: m.villa_casacasay_hero_stat_2_label(),
          value: m.villa_casacasay_hero_stat_2_value(),
        },
        {
          label: m.villa_casacasay_hero_stat_3_label(),
          value: m.villa_casacasay_hero_stat_3_value(),
        },
        {
          label: m.villa_casacasay_hero_stat_4_label(),
          value: m.villa_casacasay_hero_stat_4_value(),
        },
        {
          label: m.villa_casacasay_hero_stat_5_label(),
          value: m.villa_casacasay_hero_stat_5_value(),
        },
      ],
      price: 145,
      priceLabel: m.villa_casacasay_hero_price_label(),
    },
    ribbon: [
      m.villa_casacasay_ribbon_1(),
      m.villa_casacasay_ribbon_2(),
      m.villa_casacasay_ribbon_3(),
      m.villa_casacasay_ribbon_4(),
      m.villa_casacasay_ribbon_5(),
      m.villa_casacasay_ribbon_6(),
    ],
    about: {
      chapter: m.villa_casacasay_about_chapter(),
      titleLead: m.villa_casacasay_about_title_lead(),
      titleItalic: m.villa_casacasay_about_title_italic(),
      lead: m.villa_casacasay_about_lead(),
      body: [
        m.villa_casacasay_about_body(),
        m.villa_casacasay_about_body_2(),
        m.villa_casacasay_about_body_3(),
      ],
      features: [
        { icon: "waves-ladder", label: m.villa_casacasay_about_feature_pool() },
        { icon: "armchair", label: m.villa_casacasay_about_feature_terrace() },
        { icon: "snowflake", label: m.villa_casacasay_about_feature_ac() },
        { icon: "wifi", label: m.villa_casacasay_about_feature_wifi() },
        { icon: "shield-check", label: m.villa_casacasay_about_feature_gated() },
        { icon: "car", label: m.villa_casacasay_about_feature_parking() },
        { icon: "bed", label: m.villa_casacasay_about_feature_linens() },
        { icon: "flag", label: m.villa_casacasay_about_feature_golf() },
      ],
      meta: [
        {
          label: m.villa_casacasay_about_meta_1_label(),
          value: m.villa_casacasay_about_meta_1_value(),
        },
        {
          label: m.villa_casacasay_about_meta_2_label(),
          value: m.villa_casacasay_about_meta_2_value(),
        },
        {
          label: m.villa_casacasay_about_meta_3_label(),
          value: m.villa_casacasay_about_meta_3_value(),
        },
      ],
    },
    booking: {
      nightly: 58,
      rating: 4.92,
      reviewCount: 41,
      maxGuests: 4,
      defaultGuests: 2,
    },
    gallery: {
      chapter: m.villa_casacasay_gallery_chapter(),
      titleItalic: m.villa_casacasay_gallery_title_italic(),
      titleTail: m.villa_casacasay_gallery_title_tail(),
      description: m.villa_casacasay_gallery_description(),
      totalLabel: m.villa_casacasay_gallery_total_label(),
      tiles: [
        {
          category: "OUTDOOR",
          src: POOL_1,
          index: m.villa_casacasay_gallery_tile_1_index(),
          label: m.villa_casacasay_gallery_tile_1_index(),
          caption: m.villa_casacasay_gallery_tile_1_caption(),
          span: "wide",
        },
        {
          category: "OUTDOOR",
          src: "",
          index: m.villa_casacasay_gallery_tile_2_index(),
          label: m.villa_casacasay_gallery_tile_2_index(),
          caption: m.villa_casacasay_gallery_tile_2_caption(),
          span: "third",
          placeholder: true,
          placeholderLabel: m.villa_casacasay_gallery_tile_2_placeholder(),
        },
        {
          category: "LIVING_SPACES",
          src: "",
          index: m.villa_casacasay_gallery_tile_3_index(),
          label: m.villa_casacasay_gallery_tile_3_index(),
          caption: m.villa_casacasay_gallery_tile_3_caption(),
          span: "third",
          placeholder: true,
          placeholderLabel: m.villa_casacasay_gallery_tile_3_placeholder(),
        },
        {
          category: "BEDROOMS",
          src: "",
          index: m.villa_casacasay_gallery_tile_4_index(),
          label: m.villa_casacasay_gallery_tile_4_index(),
          caption: m.villa_casacasay_gallery_tile_4_caption(),
          span: "third",
          placeholder: true,
          placeholderLabel: m.villa_casacasay_gallery_tile_4_placeholder(),
        },
        {
          category: "KITCHEN",
          src: "",
          index: m.villa_casacasay_gallery_tile_5_index(),
          label: m.villa_casacasay_gallery_tile_5_index(),
          caption: m.villa_casacasay_gallery_tile_5_caption(),
          span: "third",
          placeholder: true,
          placeholderLabel: m.villa_casacasay_gallery_tile_5_placeholder(),
        },
      ],
      images: {
        LIVING_SPACES: [],
        BEDROOMS: [],
        KITCHEN: [],
        BATHROOMS: [],
        OUTDOOR: [
          { src: POOL_1, label: m.villa_casacasay_gallery_img_outdoor_1(), category: "OUTDOOR" },
        ],
        DINING: [],
        UTILITY: [],
      },
    },
    localArea: {
      chapter: m.villa_casacasay_local_area_chapter(),
      titleLead: m.villa_casacasay_local_area_title_lead(),
      titleItalic: m.villa_casacasay_local_area_title_italic(),
      description: m.villa_casacasay_local_area_description(),
      mainImage: POOL_1,
      stampBig: m.villa_casacasay_local_area_stamp_big(),
      stampSmall: m.villa_casacasay_local_area_stamp_small(),
      points: [
        {
          number: "№ 01",
          title: m.villa_casacasay_local_area_point_1_title(),
          description: m.villa_casacasay_local_area_point_1_description(),
          distance: m.villa_casacasay_local_area_point_1_distance(),
        },
        {
          number: "№ 02",
          title: m.villa_casacasay_local_area_point_2_title(),
          description: m.villa_casacasay_local_area_point_2_description(),
          distance: m.villa_casacasay_local_area_point_2_distance(),
        },
        {
          number: "№ 03",
          title: m.villa_casacasay_local_area_point_3_title(),
          description: m.villa_casacasay_local_area_point_3_description(),
          distance: m.villa_casacasay_local_area_point_3_distance(),
        },
        {
          number: "№ 04",
          title: m.villa_casacasay_local_area_point_4_title(),
          description: m.villa_casacasay_local_area_point_4_description(),
          distance: m.villa_casacasay_local_area_point_4_distance(),
        },
      ],
    },
    reviews: {
      chapter: m.villa_casacasay_reviews_chapter(),
      titleLead: m.villa_casacasay_reviews_title_lead(),
      titleItalic: m.villa_casacasay_reviews_title_italic(),
      description: m.villa_casacasay_reviews_description(),
      score: 4.92,
      count: 41,
      bars: [
        { label: m.villa_casacasay_reviews_bar_1_label(), pct: 98, value: "4.9" },
        { label: m.villa_casacasay_reviews_bar_2_label(), pct: 96, value: "4.8" },
        { label: m.villa_casacasay_reviews_bar_3_label(), pct: 94, value: "4.7" },
        { label: m.villa_casacasay_reviews_bar_4_label(), pct: 100, value: "5.0" },
        { label: m.villa_casacasay_reviews_bar_5_label(), pct: 96, value: "4.8" },
      ],
      entries: [
        {
          initial: "S",
          name: m.villa_casacasay_reviews_1_name(),
          meta: m.villa_casacasay_reviews_1_meta(),
          quote: m.villa_casacasay_reviews_1_quote(),
          source: m.villa_casacasay_reviews_1_source(),
          stars: 5,
        },
        {
          initial: "H",
          name: m.villa_casacasay_reviews_2_name(),
          meta: m.villa_casacasay_reviews_2_meta(),
          quote: m.villa_casacasay_reviews_2_quote(),
          source: m.villa_casacasay_reviews_2_source(),
          stars: 5,
        },
        {
          initial: "A",
          name: m.villa_casacasay_reviews_3_name(),
          meta: m.villa_casacasay_reviews_3_meta(),
          quote: m.villa_casacasay_reviews_3_quote(),
          source: m.villa_casacasay_reviews_3_source(),
          stars: 5,
        },
      ],
    },
    ctaStrip: {
      lead: m.villa_casacasay_cta_strip_lead(),
      italic: m.villa_casacasay_cta_strip_italic(),
      tail: m.villa_casacasay_cta_strip_tail(),
      button: m.villa_casacasay_cta_strip_button(),
    },
    sister: {
      chapter: m.villa_casacasay_sister_chapter(),
      titleLead: m.villa_casacasay_sister_title_lead(),
      titleItalic: m.villa_casacasay_sister_title_italic(),
      description: m.villa_casacasay_sister_description(),
      button: m.villa_casacasay_sister_button(),
      targetId: "casadana",
      image: ROOFTOP_7,
    },
  }
}

export function getVilla(id: string): VillaData | null {
  if (id === "casadana") return buildCasadana()
  if (id === "casacasay") return buildCasacasay()
  return null
}

export const VILLAS = {
  get casadana() {
    return buildCasadana()
  },
  get casacasay() {
    return buildCasacasay()
  },
} as const
