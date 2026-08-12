import CASADANA_BATHROOM from "@/assets/casadana/bathroom.jpeg"
import CASADANA_BEDROOM from "@/assets/casadana/bedroom1_1.jpeg"
import CASADANA_KITCHEN from "@/assets/casadana/kitchen1.jpeg"
import CASADANA_BG from "@/assets/casadana/rooftop7.jpeg"
import CASACASAY_BG from "@/assets/casadessy/pool1.jpeg"
import type { PropertyCardProps } from "@/components/sections/property-card"
import { m } from "@/paraglide/messages"

import { GalleryCategory } from "./gallery-categories.const"

export const properties = [
  {
    id: "casadana",
    badge: m.prop_badge_villa(),
    category: m.category_house(),
    titlePrefix: m.home_hero_title_prefix(),
    titleName: m.home_hero_title_dana(),
    subtitle: m.prop_casadana_subtitle(),
    lead: m.prop_casadana_lead(),
    description: [
      m.prop_casadana_p1(),
      m.prop_casadana_p2(),
      m.prop_casadana_p3(),
      m.prop_casadana_p4(),
    ],
    exploreLabel: m.prop_explore_dana(),
    price: { amount: 185, currency: "€" },
    rating: { score: 4.96, count: 87 },
    imageUrl: CASADANA_BG,
    imageAlt: m.prop_casadana_image_alt(),
    layout: "left",
    features: [
      { icon: "users", label: m.listing_guests({ guests: 6 }) },
      { icon: "sun", label: m.listing_rooftop() },
      { icon: "car", label: m.listing_car() },
    ],
    galleryImages: [
      {
        src: CASADANA_BG,
        alt: "Casa DaNa - Main Salon",
        label: "MAIN SALON",
        size: "large" as const,
        category: "LIVING_SPACES" as GalleryCategory,
      },
      {
        src: CASADANA_BEDROOM,
        alt: "Casa DaNa - Master Suite",
        label: "MASTER SUITE",
        size: "medium" as const,
        category: "BEDROOMS" as GalleryCategory,
      },
      {
        src: CASADANA_KITCHEN,
        alt: "Casa DaNa - Gourmet Kitchen",
        label: "GOURMET KITCHEN",
        size: "medium" as const,
        category: "KITCHEN" as GalleryCategory,
      },
      {
        src: CASADANA_BATHROOM,
        alt: "Casa DaNa - Spa Bathroom",
        label: "SPA BATHROOM",
        size: "large" as const,
        category: "BATHROOMS" as GalleryCategory,
      },
    ],
  },
  {
    id: "casacasay",
    badge: m.prop_badge_penthouse(),
    category: m.category_flat(),
    titlePrefix: m.home_hero_title_prefix(),
    titleName: m.home_hero_title_casay(),
    subtitle: m.prop_casacasay_subtitle(),
    lead: m.prop_casacasay_lead(),
    description: [
      m.prop_casacasay_p1(),
      m.prop_casacasay_p2(),
      m.prop_casacasay_p3(),
      m.prop_casacasay_p4(),
    ],
    exploreLabel: m.prop_explore_casay(),
    price: { amount: 145, currency: "€" },
    rating: { score: 4.92, count: 41 },
    imageUrl: CASACASAY_BG,
    imageAlt: m.prop_casacasay_image_alt(),
    layout: "right",
    features: [
      { icon: "users", label: m.listing_guests({ guests: 4 }) },
      { icon: "waves-ladder", label: m.listing_pool() },
      { icon: "armchair", label: m.listing_exterior() },
    ],
    galleryImages: [
      {
        src: CASACASAY_BG,
        alt: "Casa CasAy - Pool",
        label: "POOL VIEW",
        size: "large" as const,
        category: "OUTDOOR" as GalleryCategory,
      },
    ],
  },
] satisfies Array<PropertyCardProps>
